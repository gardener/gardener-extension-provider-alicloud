// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package infraflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/gardener/gardener/pkg/utils/flow"

	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
)

const (
	// IdentifierZoneRetirementCompleted marks a retired zone whose managed resources have been handled.
	IdentifierZoneRetirementCompleted = "RetirementCompleted"
	// ZoneRetirementCompletedValue is the persisted value of IdentifierZoneRetirementCompleted.
	ZoneRetirementCompletedValue = "true"
)

// runZoneRetirementCleanup retires zones serially so shared egress resources are never modified concurrently.
func (c *FlowContext) runZoneRetirementCleanup(ctx context.Context, targets []zoneRetirementTarget) error {
	for _, target := range targets {
		if c.isZoneRetirementCompleted(target.ZoneName) {
			continue
		}
		if err := c.runZoneRetirementSubgraph(ctx, target); err != nil {
			return err
		}
	}
	return nil
}

func (c *FlowContext) runZoneRetirementSubgraph(ctx context.Context, target zoneRetirementTarget) error {
	g := flow.NewGraph("Alicloud zone retirement: " + target.ZoneName)

	// Remove dependencies that prevent the target VSwitch from being deleted.
	deleteSNAT := c.AddTask(
		g,
		"delete target SNAT",
		c.deleteRetirementSNAT(target),
		Timeout(defaultTimeout),
	)
	unassociateRouteTable := c.AddTask(
		g,
		"unassociate target route table",
		c.unassociateRetirementRouteTable(target),
		Timeout(defaultTimeout), Dependencies(deleteSNAT),
	)
	deleteVSwitch := c.AddTask(
		g,
		"delete target VSwitch",
		c.deleteRetirementVSwitch(target),
		Timeout(defaultTimeout), Dependencies(unassociateRouteTable),
	)

	// EIPs are created for a zone but are not bound to its VSwitch and may be reused by another zone.
	cleanupEIPAssociation := c.AddTask(
		g,
		"cleanup target EIP association",
		c.cleanupRetirementEIPAssociations(target),
		Timeout(defaultTimeout), Dependencies(deleteVSwitch),
	)
	releaseManagedEIP := c.AddTask(
		g,
		"release target managed EIP",
		c.releaseRetirementManagedEIP(target),
		Timeout(defaultTimeout), Dependencies(cleanupEIPAssociation),
	)

	// Publish Completed only after all cloud resources have reached their desired state.
	_ = c.AddTask(
		g,
		"finalize zone retirement state",
		c.finalizeZoneRetirementState(target),
		Dependencies(releaseManagedEIP),
	)

	if err := g.Compile().Run(ctx, flow.Opts{Log: c.Log}); err != nil {
		return flow.Causes(err)
	}
	return nil
}

func (c *FlowContext) deleteRetirementSNAT(target zoneRetirementTarget) flow.TaskFn {
	return func(ctx context.Context) error {
		if !target.ManagedVSwitch || target.VSwitchID == "" {
			return nil
		}
		ngwID, err := c.retirementNatGatewayID(target.ZoneName)
		if err != nil {
			return err
		}
		entries, err := c.actor.FindSNatEntriesByNatGateway(ctx, ngwID)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.VSwitchId != target.VSwitchID {
				continue
			}
			if err := c.actor.DeleteSNatEntry(ctx, entry.SnatEntryId, entry.SnatTableId); err != nil {
				return err
			}
		}
		return nil
	}
}

// cleanupRetirementEIPAssociations runs after the target VSwitch's SNAT entries are deleted.
// Any remaining SNAT entry using an EIP therefore belongs outside the retiring zone.
func (c *FlowContext) cleanupRetirementEIPAssociations(target zoneRetirementTarget) flow.TaskFn {
	return func(ctx context.Context) error {
		if target.ManagedEIPID != "" {
			if err := c.cleanupRetirementEIPAssociation(ctx, target, target.ManagedEIPID, true); err != nil {
				return err
			}
		}
		if target.UserEIPID == "" || target.UserEIPID == target.ManagedEIPID {
			return nil
		}
		return c.cleanupRetirementEIPAssociation(ctx, target, target.UserEIPID, false)
	}
}

func (c *FlowContext) cleanupRetirementEIPAssociation(ctx context.Context, target zoneRetirementTarget, eipID string, managed bool) error {
	if c.activeZoneReferencesEIP(eipID) {
		return nil
	}
	eip, err := c.actor.GetEIP(ctx, eipID)
	if err != nil {
		return err
	}
	if eip == nil {
		return nil
	}
	if eip.Status == nil {
		return fmt.Errorf("EIP %s has no status", eipID)
	}
	if *eip.Status == "Available" {
		return nil
	}
	if *eip.Status != "InUse" {
		return fmt.Errorf("EIP %s has unsupported status %s", eipID, *eip.Status)
	}
	if eip.InstanceId == nil || eip.InstanceType == nil {
		return fmt.Errorf("EIP %s is in use without association details", eipID)
	}
	ngwID, err := c.retirementNatGatewayID(target.ZoneName)
	if err != nil {
		return err
	}
	if *eip.InstanceId != ngwID {
		if managed {
			return fmt.Errorf("managed EIP %s is associated with unexpected resource %s", eipID, *eip.InstanceId)
		}
		return nil
	}
	entries, err := c.actor.FindSNatEntriesByNatGateway(ctx, ngwID)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if snatUsesIPAddress(entry.IpAddress, eip.IpAddress) {
			// The EIP is shared with another zone (or with a user-managed SNAT entry).
			// Keep the NGW association; this zone can still finish retirement.
			return nil
		}
	}
	return c.actor.UnAssociateEIP(ctx, eip)
}

// releaseRetirementManagedEIP releases only an unused EIP without an explicit zone reference.
// A deferred release keeps the managed EIP state so a later Shoot deletion can still reclaim it.
// Remaining SNAT usage does not block this zone's retirement.
func (c *FlowContext) releaseRetirementManagedEIP(target zoneRetirementTarget) flow.TaskFn {
	return func(ctx context.Context) error {
		if target.ManagedEIPID == "" {
			return nil
		}
		zone := c.getZoneChild(target.ZoneName)
		if c.zoneConfigReferencesEIP(target.ManagedEIPID) {
			return nil
		}
		eip, err := c.actor.GetEIP(ctx, target.ManagedEIPID)
		if err != nil {
			return err
		}
		if eip == nil {
			zone.SetAsDeleted(IdentifierZoneNATGWElasticIP)
			return nil
		}
		if eip.Status == nil {
			return fmt.Errorf("managed EIP %s has no status", target.ManagedEIPID)
		}
		if *eip.Status == "InUse" {
			if eip.InstanceId == nil || eip.InstanceType == nil {
				return fmt.Errorf("managed EIP %s is in use without association details", target.ManagedEIPID)
			}
			ngwID, err := c.retirementNatGatewayID(target.ZoneName)
			if err != nil {
				return err
			}
			if *eip.InstanceId != ngwID {
				return fmt.Errorf("managed EIP %s is associated with unexpected resource %s", target.ManagedEIPID, *eip.InstanceId)
			}
			entries, err := c.actor.FindSNatEntriesByNatGateway(ctx, ngwID)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if snatUsesIPAddress(entry.IpAddress, eip.IpAddress) {
					return nil
				}
			}
			return fmt.Errorf("managed EIP %s is still associated after its SNAT usage was removed", target.ManagedEIPID)
		}
		if *eip.Status != "Available" {
			return fmt.Errorf("managed EIP %s has unsupported status %s", target.ManagedEIPID, *eip.Status)
		}
		if err := c.actor.DeleteEIP(ctx, target.ManagedEIPID); err != nil {
			return err
		}
		zone.SetAsDeleted(IdentifierZoneNATGWElasticIP)
		return nil
	}
}

func (c *FlowContext) unassociateRetirementRouteTable(target zoneRetirementTarget) flow.TaskFn {
	return func(ctx context.Context) error {
		if !target.ManagedVSwitch || target.VSwitchID == "" || !c.useCustomRouteTable() {
			return nil
		}
		vSwitch, err := c.actor.GetVSwitch(ctx, target.VSwitchID)
		if err != nil {
			return err
		}
		if vSwitch == nil {
			return nil
		}
		routeTableID := c.state.Get(IdentifierRouteTable)
		if routeTableID == nil {
			return fmt.Errorf("cannot unassociate VSwitch %s: IdentifierRouteTable is nil", target.VSwitchID)
		}
		routeTable, err := c.actor.GetRouteTable(ctx, *routeTableID)
		if err != nil {
			return err
		}
		if routeTable == nil {
			return nil
		}
		if !contains(routeTable.VSwitchIds, target.VSwitchID) {
			return nil
		}
		return c.actor.UnassociateRouteTable(ctx, *routeTableID, target.VSwitchID)
	}
}

// deleteRetirementVSwitch relies on the graph dependencies to remove target SNAT entries and
// the custom route table association first. The target identity was verified during preflight.
func (c *FlowContext) deleteRetirementVSwitch(target zoneRetirementTarget) flow.TaskFn {
	return func(ctx context.Context) error {
		if !target.ManagedVSwitch {
			return nil
		}
		child := c.getZoneChild(target.ZoneName)
		if target.VSwitchID != "" {
			vSwitch, err := c.actor.GetVSwitch(ctx, target.VSwitchID)
			if err != nil {
				return err
			}
			if vSwitch != nil {
				if err := c.actor.DeleteVSwitch(ctx, target.VSwitchID); err != nil {
					return err
				}
			}
		}
		child.SetAsDeleted(IdentifierZoneVSwitch)
		return nil
	}
}

// finalizeZoneRetirementState publishes the final state only after both the VSwitch and its managed residuals are handled.
func (c *FlowContext) finalizeZoneRetirementState(target zoneRetirementTarget) flow.TaskFn {
	return func(_ context.Context) error {
		zone := c.getZoneChild(target.ZoneName)
		if target.ManagedVSwitch {
			zone.SetAsDeleted(ZoneNATGWElasticIPAddress)
		}
		zone.Set(IdentifierZoneRetirementCompleted, ZoneRetirementCompletedValue)
		return nil
	}
}

func (c *FlowContext) activeZoneReferencesEIP(eipID string) bool {
	for i := range c.config.Networks.Zones {
		zone := &c.config.Networks.Zones[i]
		if zone.Retired {
			continue
		}
		if zone.NatGateway != nil && zone.NatGateway.EIPAllocationID != nil && *zone.NatGateway.EIPAllocationID == eipID {
			return true
		}
	}
	return false
}

func (c *FlowContext) zoneConfigReferencesEIP(eipID string) bool {
	for i := range c.config.Networks.Zones {
		zone := &c.config.Networks.Zones[i]
		if zone.NatGateway != nil && zone.NatGateway.EIPAllocationID != nil && *zone.NatGateway.EIPAllocationID == eipID {
			return true
		}
	}
	return false
}

func (c *FlowContext) retirementNatGatewayID(zoneName string) (string, error) {
	id := c.state.Get(IdentifierNatGateway)
	if id == nil {
		return "", fmt.Errorf("cannot clean zone %s: IdentifierNatGateway is nil", zoneName)
	}
	return *id, nil
}

func (c *FlowContext) isZoneRetirementCompleted(zoneName string) bool {
	completed := c.getZoneChild(zoneName).Get(IdentifierZoneRetirementCompleted)
	return completed != nil && *completed == ZoneRetirementCompletedValue
}

func snatUsesIPAddress(value, address string) bool {
	for _, item := range strings.Split(value, ",") {
		if strings.TrimSpace(item) == address {
			return true
		}
	}
	return false
}
