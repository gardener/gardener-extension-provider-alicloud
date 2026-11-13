// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package infraflow

import (
	"context"
	"fmt"

	aliapi "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	alicloudhelper "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
)

// ZoneRetirementObserver performs Seed-side read-only checks needed before a VSwitch can be retired.
type ZoneRetirementObserver interface {
	EnsureVSwitchUnused(ctx context.Context, namespace, vSwitchID string) error
}

// zoneRetirementTarget identifies the resources belonging to one disabled zone.
type zoneRetirementTarget struct {
	ZoneName string
	// VSwitchID is empty when neither state nor Alicloud contains a VSwitch owned by this Shoot.
	VSwitchID      string
	ManagedVSwitch bool
	ManagedEIPID   string
	UserEIPID      string
}

func (c *FlowContext) buildZoneRetirementTargets(ctx context.Context) ([]zoneRetirementTarget, error) {
	var targets []zoneRetirementTarget
	for i := range c.config.Networks.Zones {
		zone := &c.config.Networks.Zones[i]
		if !zone.Disabled {
			continue
		}
		if c.isZoneRetirementCompleted(zone.Name) {
			continue
		}
		if c.retirementObserver == nil {
			return nil, fmt.Errorf("zone retirement observer is nil")
		}

		target, err := c.buildZoneRetirementTarget(ctx, zone)
		if err != nil {
			return nil, err
		}
		if err := c.ensureWorkerSpecDoesNotUseZone(zone.Name); err != nil {
			return nil, err
		}
		if target.VSwitchID != "" {
			if err := c.retirementObserver.EnsureVSwitchUnused(ctx, c.namespace, target.VSwitchID); err != nil {
				return nil, fmt.Errorf("zone %s is not ready for retirement: %w", zone.Name, err)
			}
		}
		targets = append(targets, target)
	}

	if len(targets) == 0 {
		return nil, nil
	}
	if err := c.ensureManagedNatGatewayOutsideTargets(ctx, targets); err != nil {
		return nil, err
	}
	return targets, nil
}

func (c *FlowContext) buildZoneRetirementTarget(ctx context.Context, zone *aliapi.Zone) (zoneRetirementTarget, error) {
	zoneName := zone.Name
	isBYO := c.isBYOInfrastructure()
	target := zoneRetirementTarget{ZoneName: zoneName, ManagedVSwitch: !isBYO}

	if isBYO {
		if zone.WorkersVSwitchID == nil {
			return zoneRetirementTarget{}, fmt.Errorf("cannot retire BYO zone %s: workersVSwitchID is nil", zoneName)
		}
		target.VSwitchID = *zone.WorkersVSwitchID
		return target, nil
	}

	vpcID := c.state.Get(IdentifierVPC)
	if vpcID == nil {
		return zoneRetirementTarget{}, fmt.Errorf("cannot retire zone %s: IdentifierVPC is nil", zoneName)
	}
	cidrBlock := zoneWorkerCIDR(zone.Workers, zone.Worker)
	vSwitchID, err := c.resolveManagedRetirementVSwitch(ctx, zoneName, *vpcID, cidrBlock)
	if err != nil {
		return zoneRetirementTarget{}, err
	}
	target.VSwitchID = vSwitchID

	zoneChild := c.getZoneChild(zoneName)
	managedEIPID := zoneChild.Get(IdentifierZoneNATGWElasticIP)
	if zone.NatGateway != nil && zone.NatGateway.EIPAllocationID != nil {
		target.UserEIPID = *zone.NatGateway.EIPAllocationID
	}
	if managedEIPID != nil {
		target.ManagedEIPID = *managedEIPID
		if err := c.validateManagedRetirementEIP(ctx, zoneName, target.ManagedEIPID); err != nil {
			return zoneRetirementTarget{}, err
		}
	}

	return target, nil
}

func (c *FlowContext) resolveManagedRetirementVSwitch(ctx context.Context, zoneName, vpcID, cidrBlock string) (string, error) {
	expectedName := c.namespace + "-" + zoneName + "-vsw"
	zoneState := c.getZoneChild(zoneName)
	if zoneState.IsAlreadyDeleted(IdentifierZoneVSwitch) {
		return "", nil
	}
	recordedID := zoneState.Get(IdentifierZoneVSwitch)
	if recordedID != nil {
		found, err := c.actor.ListVSwitches(ctx, []string{*recordedID})
		if err != nil {
			return "", fmt.Errorf("get recorded VSwitch %s for retiring zone %s: %w", *recordedID, zoneName, err)
		}
		if len(found) == 0 {
			// Preserve the exact recorded ID even if the VSwitch was already removed externally.
			return *recordedID, nil
		}
		if err := c.validateManagedRetirementVSwitch(found[0], zoneName, vpcID, cidrBlock, expectedName); err != nil {
			return "", err
		}
		return found[0].VSwitchId, nil
	}

	found, err := c.actor.FindVSwitchesByVPC(ctx, vpcID)
	if err != nil {
		return "", fmt.Errorf("find VSwitch for retiring zone %s in VPC %s: %w", zoneName, vpcID, err)
	}
	for _, candidate := range found {
		if candidate.ZoneId != zoneName || candidate.CidrBlock != cidrBlock {
			continue
		}
		if !c.isOwnedByCurrentShoot(candidate.Tags, candidate.Name, expectedName) {
			continue
		}
		return candidate.VSwitchId, nil
	}
	return "", nil
}

func (c *FlowContext) validateManagedRetirementVSwitch(vSwitch *aliclient.VSwitch, zoneName, vpcID, cidrBlock, expectedName string) error {
	if vSwitch.VpcId == nil || *vSwitch.VpcId != vpcID {
		return fmt.Errorf("VSwitch %s for retiring zone %s is not in expected VPC %s", vSwitch.VSwitchId, zoneName, vpcID)
	}
	if vSwitch.ZoneId != zoneName || vSwitch.CidrBlock != cidrBlock {
		return fmt.Errorf("VSwitch %s identity does not match retiring zone %s and CIDR %s", vSwitch.VSwitchId, zoneName, cidrBlock)
	}
	if !c.isOwnedByCurrentShoot(vSwitch.Tags, vSwitch.Name, expectedName) {
		return fmt.Errorf("VSwitch %s ownership cannot be verified for shoot %s", vSwitch.VSwitchId, c.namespace)
	}
	return nil
}

func (c *FlowContext) validateManagedRetirementEIP(ctx context.Context, zoneName, eipID string) error {
	eip, err := c.actor.GetEIP(ctx, eipID)
	if err != nil {
		return fmt.Errorf("get managed EIP %s for retiring zone %s: %w", eipID, zoneName, err)
	}
	if eip == nil {
		return nil
	}
	expectedName := ""
	if suffix := c.getZoneChild(zoneName).Get(IdentifierZoneSuffix); suffix != nil {
		expectedName = fmt.Sprintf("%s-eip-natgw-%s", c.namespace, *suffix)
	}
	if !c.isOwnedByCurrentShoot(eip.Tags, eip.Name, expectedName) {
		return fmt.Errorf("EIP %s ownership cannot be verified for shoot %s", eipID, c.namespace)
	}
	return nil
}

func (c *FlowContext) ensureManagedNatGatewayOutsideTargets(ctx context.Context, targets []zoneRetirementTarget) error {
	if !alicloudhelper.IsManagedNATGatewayConfig(c.config) {
		return nil
	}
	natGatewayID := c.state.Get(IdentifierNatGateway)
	if natGatewayID == nil {
		return fmt.Errorf("cannot verify zone retirement: IdentifierNatGateway is nil")
	}
	natGateway, err := c.actor.GetNatGateway(ctx, *natGatewayID)
	if err != nil {
		return fmt.Errorf("get managed NatGateway %s before zone retirement: %w", *natGatewayID, err)
	}
	if natGateway == nil || natGateway.VswitchId == nil || *natGateway.VswitchId == "" {
		return fmt.Errorf("cannot determine VSwitch of managed NatGateway %s", *natGatewayID)
	}
	for _, target := range targets {
		if target.VSwitchID == *natGateway.VswitchId {
			return fmt.Errorf("cannot retire zone %s: managed NatGateway %s is located in target VSwitch %s", target.ZoneName, *natGatewayID, target.VSwitchID)
		}
	}
	return nil
}

func (c *FlowContext) ensureWorkerSpecDoesNotUseZone(zoneName string) error {
	if c.cluster == nil || c.cluster.Shoot == nil {
		return fmt.Errorf("cluster or Shoot is nil")
	}
	for _, worker := range c.cluster.Shoot.Spec.Provider.Workers {
		for _, workerZone := range worker.Zones {
			if workerZone == zoneName {
				return fmt.Errorf("worker pool %s still references retiring zone %s", worker.Name, zoneName)
			}
		}
	}
	return nil
}

func zoneWorkerCIDR(workers, worker string) string {
	if workers != "" {
		return workers
	}
	return worker
}
