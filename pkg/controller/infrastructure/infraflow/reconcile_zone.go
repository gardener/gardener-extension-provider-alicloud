// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infraflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gardener/gardener/pkg/utils/flow"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
)

func (c *FlowContext) ensureZones(ctx context.Context) error {
	log := c.LogFromContext(ctx)
	log.Info("begin ensure zones")
	g := flow.NewGraph("Alicloud infrastructure : zones")

	eipIntenetChargeType := c.getEipInternetChargeType(ctx)

	for _, zone := range c.config.Networks.Zones {
		c.addZoneReconcileTasks(g, zone.Name, eipIntenetChargeType)
	}

	f := g.Compile()
	if err := f.Run(ctx, flow.Opts{Log: c.Log}); err != nil {
		return flow.Causes(err)
	}
	return nil
}

func (c *FlowContext) getEipInternetChargeType(ctx context.Context) string {
	var eipId *string
	eipId = nil
	for _, zone := range c.config.Networks.Zones {
		if zone.NatGateway != nil && zone.NatGateway.EIPAllocationID != nil {
			eipId = zone.NatGateway.EIPAllocationID
			break
		}
	}
	if eipId == nil {
		return client.DefaultInternetChargeType
	}
	provideEip, _ := c.actor.GetEIP(ctx, *eipId)
	if provideEip != nil {
		return provideEip.InternetChargeType
	}
	return client.DefaultInternetChargeType
}
func (c *FlowContext) addZoneReconcileTasks(g *flow.Graph, zoneName, eipIntenetChargeType string) {
	ensureElasticIP := c.AddTask(g, "ensure elastic IP "+zoneName,
		c.ensureElasticIP(zoneName, eipIntenetChargeType),
		Timeout(defaultLongTimeout))
	ensureEipAssociation := c.AddTask(g, "ensure eip association "+zoneName,
		c.ensureEipAssociation(zoneName),
		Timeout(defaultLongTimeout), Dependencies(ensureElasticIP))
	_ = c.AddTask(g, "ensure snat entry "+zoneName,
		c.ensureSnatEntry(zoneName),
		Timeout(defaultLongTimeout), Dependencies(ensureEipAssociation))
}

func (c *FlowContext) ensureSnatEntry(zoneName string) flow.TaskFn {
	return func(ctx context.Context) error {
		log := c.LogFromContext(ctx)
		log.Info("ensureSnatEntry", "zoneName", zoneName)
		zone := c.getZoneConfig(zoneName)
		child := c.getZoneChild(zoneName)
		managed_eipId := child.Get(IdentifierZoneNATGWElasticIP)
		ngwId := c.state.Get(IdentifierNatGateway)
		vswitchId := child.Get(IdentifierZoneVSwitch)
		if vswitchId == nil {
			return fmt.Errorf("IdentifierZoneVSwitch is nil")
		}
		eipId := managed_eipId
		if zone.NatGateway != nil && zone.NatGateway.EIPAllocationID != nil {
			eipId = zone.NatGateway.EIPAllocationID
		}
		if eipId == nil {
			return fmt.Errorf("no Eip exist @ zone %s", zoneName)
		}
		eip, err := c.actor.GetEIP(ctx, *eipId)
		if err != nil {
			return err
		}
		if eip == nil {
			return fmt.Errorf("not find the recorded EIP %s", *eipId)
		}
		if ngwId == nil {
			return fmt.Errorf("IdentifierNatGateway is nil")
		}
		ngw, err := c.actor.GetNatGateway(ctx, *ngwId)
		if err != nil {
			return err
		}
		if ngw == nil {
			return fmt.Errorf("not find the recorded NATGATEWAY %s", *ngwId)
		}

		zoneSuffix := c.getZoneSuffix(zone.Name)
		snatSuffix := fmt.Sprintf("snat-%s", zoneSuffix)
		var desired []*aliclient.SNATEntry
		for _, snateTableId := range ngw.SNATTableIDs {
			desired = append(desired, &aliclient.SNATEntry{
				Name:         c.namespace + "-" + snatSuffix + "-" + snateTableId,
				NatGatewayId: *ngwId,
				VSwitchId:    *vswitchId,
				IpAddress:    eip.IpAddress,
				SnatTableId:  snateTableId,
			})
		}

		current, err := c.getCurrentSnatEntryForZone(ctx, zoneName)
		if err != nil {
			return err
		}
		toBeDeleted, toBeCreated, toBeChecked := diffByID(desired, current, func(item *aliclient.SNATEntry) string {
			return item.SnatTableId + "-" + item.VSwitchId
		})
		for _, entry := range toBeDeleted {
			if err := c.actor.DeleteSNatEntry(ctx, entry.SnatEntryId, entry.SnatTableId); err != nil {
				return err
			}
		}
		for _, desired := range toBeCreated {
			created, err := c.actor.CreateSNatEntry(ctx, desired)
			if err != nil {
				return err
			}
			if created == nil {
				return fmt.Errorf("failed to create SNAT entry")
			}
			_, _ = c.updater.UpdateSNATEntry(ctx, desired, created)
		}
		toUnAssociateEIPs := sets.New[string]()
		for _, item := range toBeChecked {
			if item.desired.IpAddress != item.current.IpAddress {
				toUnAssociateEIPs.Insert(item.current.IpAddress)

				waiter := informOnWaiting(log, 5*time.Second, "still deleting snate entry ...", "SnatEntryId", item.current.SnatEntryId, "SnatTableId", item.current.SnatTableId)
				err := c.actor.DeleteSNatEntry(ctx, item.current.SnatEntryId, item.current.SnatTableId)
				waiter.Done(err)
				if err != nil {
					return err
				}

				created, err := c.actor.CreateSNatEntry(ctx, item.desired)
				if err != nil {
					return err
				}
				if created == nil {
					return fmt.Errorf("failed to create SNAT entry")
				}

				_, _ = c.updater.UpdateSNATEntry(ctx, item.desired, created)
			} else {
				_, _ = c.updater.UpdateSNATEntry(ctx, item.desired, item.current)
			}
		}

		for ipAddress := range toUnAssociateEIPs {
			the_eip, err := c.actor.GetEIPByAddress(ctx, ipAddress)
			if err != nil {
				return err
			}
			if the_eip != nil && *the_eip.Status == "InUse" {
				if err := c.actor.UnAssociateEIP(ctx, the_eip); err != nil {
					return err
				}
			}
		}

		if managed_eipId != nil && eipId != managed_eipId {
			managed_eip, err := c.actor.GetEIP(ctx, *managed_eipId)
			if err != nil {
				return err
			}

			if managed_eip != nil {
				log := c.LogFromContext(ctx)
				log.Info("deleting...", "AllocationId", managed_eip.EipId)
				waiter := informOnWaiting(log, 5*time.Second, "still deleting...", "AllocationId", managed_eip.EipId)
				err = c.actor.DeleteEIP(ctx, managed_eip.EipId)
				waiter.Done(err)
				if err != nil {
					return err
				}
			}
			child.SetAsDeleted(IdentifierZoneNATGWElasticIP)
			if err := c.PersistState(ctx, true); err != nil {
				return err
			}
		}

		return nil
	}
}

func (c *FlowContext) getCurrentSnatEntryForZone(ctx context.Context, zoneName string) ([]*aliclient.SNATEntry, error) {
	child := c.getZoneChild(zoneName)
	vswitchId := child.Get(IdentifierZoneVSwitch)
	ngwId := c.state.Get(IdentifierNatGateway)
	entryList := []*aliclient.SNATEntry{}
	if ngwId == nil || vswitchId == nil {
		return entryList, nil
	}
	entries, err := c.actor.FindSNatEntriesByNatGateway(ctx, *ngwId)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.VSwitchId == *vswitchId {
			entryList = append(entryList, entry)
		}
	}
	return entryList, nil
}

func (c *FlowContext) ensureEipAssociation(zoneName string) flow.TaskFn {
	return func(ctx context.Context) error {
		log := c.LogFromContext(ctx)

		zone := c.getZoneConfig(zoneName)
		child := c.getZoneChild(zoneName)
		eipId := child.Get(IdentifierZoneNATGWElasticIP)
		ngwId := c.state.Get(IdentifierNatGateway)
		if ngwId == nil {
			return fmt.Errorf("IdentifierNatGateway is nil")
		}

		if zone.NatGateway != nil && zone.NatGateway.EIPAllocationID != nil {
			eipId = zone.NatGateway.EIPAllocationID
		}
		if eipId == nil {
			return fmt.Errorf("no Eip exist @ zone %s", zoneName)
		}
		log.Info("ensure eip associated to zone for NatGateway", "eipId", *eipId, "zoneName", zoneName, "ngwId", *ngwId)
		eip, err := c.actor.GetEIP(ctx, *eipId)
		if err != nil {
			return err
		}
		if eip == nil {
			return fmt.Errorf("not find the recorded EIP %s", *eipId)
		}
		switch *eip.Status {
		case "Available":
			// association
			err := c.actor.AssociateEIP(ctx, *eipId, *ngwId, "Nat")
			if err != nil {
				return err
			}
		case "InUse":
			if *eip.InstanceId != *ngwId {
				return fmt.Errorf("the eip %s is not associated to natgateway %s", *eipId, *ngwId)
			}
		default:
			return fmt.Errorf(" eip %s status %s not allowed", *eipId, *eip.Status)
		}
		return nil
	}
}

func (c *FlowContext) ensureElasticIP(zoneName, eipIntenetChargeType string) flow.TaskFn {
	return func(ctx context.Context) error {
		zone := c.getZoneConfig(zoneName)
		if zone == nil {
			return fmt.Errorf("can not get zone config for %s", zoneName)
		}
		log := c.LogFromContext(ctx)
		child := c.getZoneChild(zone.Name)
		if zone.NatGateway != nil && zone.NatGateway.EIPAllocationID != nil {
			eipId := *zone.NatGateway.EIPAllocationID
			log.Info("using configured EIP", "eipId", eipId)
			current, err := c.actor.GetEIP(ctx, eipId)
			if err != nil {
				return err
			}
			if current == nil {
				return fmt.Errorf("configured EIP %s has not been found", eipId)
			}
			child.Set(ZoneNATGWElasticIPAddress, current.IpAddress)
			return c.PersistState(ctx, true)
		}
		zoneSuffix := c.getZoneSuffix(zone.Name)
		eipSuffix := fmt.Sprintf("eip-natgw-%s", zoneSuffix)

		desired := &aliclient.EIP{
			Name:               c.namespace + "-" + eipSuffix,
			Tags:               c.commonTagsWithSuffix(eipSuffix),
			Bandwidth:          "100",
			InternetChargeType: eipIntenetChargeType,
		}
		current, err := findExisting(ctx, child.Get(IdentifierZoneNATGWElasticIP), desired.Tags, c.actor.GetEIP, c.actor.FindEIPsByTags)
		if err != nil {
			return err
		}

		if current != nil {
			child.Set(IdentifierZoneNATGWElasticIP, current.EipId)
			child.Set(ZoneNATGWElasticIPAddress, current.IpAddress)
			if _, err := c.updater.UpdateEIP(ctx, desired, current); err != nil {
				return err
			}
		} else {
			log.Info("creating eip ...")
			created, err := c.actor.CreateEIP(ctx, desired)
			if err != nil {
				return err
			}
			if created == nil {
				return fmt.Errorf("failed to create EIP")
			}
			child.Set(IdentifierZoneNATGWElasticIP, created.EipId)
			child.Set(ZoneNATGWElasticIPAddress, created.IpAddress)
			if _, err := c.updater.UpdateEIP(ctx, desired, created); err != nil {
				return err
			}
		}
		return c.PersistState(ctx, true)
	}
}

func (c *FlowContext) ensureVSwitches(ctx context.Context) error {
	vpcId := c.state.Get(IdentifierVPC)
	if vpcId == nil {
		return fmt.Errorf("IdentifierVPC is nil")
	}
	log := c.LogFromContext(ctx)
	var desired []*aliclient.VSwitch
	for _, zone := range c.config.Networks.Zones {
		zoneSuffix := c.getZoneSuffix(zone.Name)
		workerSuffix := fmt.Sprintf("nodes-%s", zoneSuffix)
		cidrBlock := zone.Workers
		if cidrBlock == "" {
			cidrBlock = zone.Worker
		}
		desired = append(desired,
			&aliclient.VSwitch{
				Name:      c.namespace + "-" + zone.Name + "-vsw",
				CidrBlock: cidrBlock,
				VpcId:     vpcId,
				Tags:      c.commonTagsWithSuffix(workerSuffix),
				ZoneId:    zone.Name,
			})
	}

	current, err := c.collectExistingVSwitches(ctx)
	if err != nil {
		return err
	}
	vpc_vsw, err := c.actor.FindVSwitchesByVPC(ctx, *vpcId)
	if err != nil {
		return err
	}

	toBeDeleted, toBeCreated, toBeChecked := diffByID_Ex(desired, current, vpc_vsw, func(item *aliclient.VSwitch) string {
		return item.ZoneId + "-" + item.CidrBlock
	})

	if len(toBeDeleted) > 0 && !c.canDelete {
		var details []string
		for _, vsw := range toBeDeleted {
			zone_name := getZoneName(vsw)
			vswitch_id := vsw.VSwitchId
			details = append(details, fmt.Sprintf("zone: %s, vswitch: %s", zone_name, vswitch_id))
		}
		return fmt.Errorf("protected: attempt to DeleteZoneByVSwitches during reconcile. Details: %s", strings.Join(details, "; "))
	}
	if err := c.DeleteZoneByVSwitches(ctx, toBeDeleted); err != nil {
		return err
	}

	for _, desired := range toBeCreated {
		log.Info("creating vswitch ...")
		created, err := c.actor.CreateVSwitch(ctx, desired)
		if err != nil {
			return err
		}
		if created == nil {
			return fmt.Errorf("failed to create vswitch")
		}
		c.state.GetChild(ChildIdZones).GetChild(desired.ZoneId).Set(IdentifierZoneVSwitch, created.VSwitchId)
		_, err = c.updater.UpdateVSwitch(ctx, desired, created)
		if err != nil {
			return err
		}
	}
	for _, vsw := range toBeChecked {
		c.state.GetChild(ChildIdZones).GetChild(vsw.current.ZoneId).Set(IdentifierZoneVSwitch, vsw.current.VSwitchId)
		_, err = c.updater.UpdateVSwitch(ctx, vsw.desired, vsw.current)
		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteZoneByVSwitches is called to delete zone per vswitch
func (c *FlowContext) DeleteZoneByVSwitches(ctx context.Context, toBeDeleted []*aliclient.VSwitch) error {
	// Check if toBeDeleted is empty
	if len(toBeDeleted) == 0 {
		return nil // Return immediately if there is nothing to delete
	}
	needDeleteNatGateway := c.config.Networks.VPC.ID == nil || (c.config.Networks.VPC.GardenerManagedNATGateway != nil && *c.config.Networks.VPC.GardenerManagedNATGateway)
	g := flow.NewGraph("Alicloud infrastructure deletion: zones")

	toBeDeletedZones := sets.NewString()
	vswitchIds := []string{}
	for _, vsw := range toBeDeleted {
		toBeDeletedZones.Insert(getZoneName(vsw))
		vswitchIds = append(vswitchIds, vsw.VSwitchId)
	}
	dependencies := []flow.TaskIDer{}
	for zoneName := range toBeDeletedZones {
		taskID := c.addZoneDeletionTasks(g, zoneName)
		if taskID != nil {
			dependencies = append(dependencies, taskID)
		}
	}

	deleteNatGateway := c.AddTask(g, "delete managed NatGateway",
		c.deleteNatGatewayInVSwitches(vswitchIds),
		DoIf(needDeleteNatGateway && c.hasNatGateway()), Timeout(defaultLongTimeout), Dependencies(dependencies...))

	for _, vsw := range toBeDeleted {
		c.AddTask(g, "delete vswitch resource "+getZoneName(vsw),
			c.deleteVSwitch(vsw),
			Timeout(defaultTimeout), Dependencies(deleteNatGateway))
	}

	f := g.Compile()
	if err := f.Run(ctx, flow.Opts{Log: c.Log}); err != nil {
		return flow.Causes(err)
	}

	zones := c.state.GetChild(ChildIdZones)
	for zoneName := range toBeDeletedZones {
		zones.CleanChild(zoneName)
	}

	return c.PersistState(ctx, true)
}

func (c *FlowContext) addZoneDeletionTasks(g *flow.Graph, zoneName string) flow.TaskIDer {
	deleteSNatEntryForZone := c.AddTask(g, "delete snat entry for zone "+zoneName,
		c.deleteSNatEntryForZone(zoneName),
		Timeout(defaultTimeout))

	deleteEipAssociation := c.AddTask(g, "delete eip association "+zoneName,
		c.deleteEipAssociation(zoneName),
		Timeout(defaultTimeout), Dependencies(deleteSNatEntryForZone))

	deleteElasticIP := c.AddTask(g, "delete elastic IP "+zoneName,
		c.deleteElasticIP(zoneName),
		Timeout(defaultTimeout), Dependencies(deleteEipAssociation))

	return deleteElasticIP
}

func (c *FlowContext) deleteSNatEntryForZone(zoneName string) flow.TaskFn {
	return func(ctx context.Context) error {
		log := c.LogFromContext(ctx)
		log.Info("deleting snate entry for zone ...", "zoneName", zoneName)

		current, err := c.getCurrentSnatEntryForZone(ctx, zoneName)
		if err != nil {
			return err
		}
		toUnAssociateEIPs := sets.New[string]()
		for _, entry := range current {
			toUnAssociateEIPs.Insert(entry.IpAddress)
			waiter := informOnWaiting(log, 5*time.Second, "still deleting snate entry ...", "SnatEntryId", entry.SnatEntryId, "SnatTableId", entry.SnatTableId)
			err := c.actor.DeleteSNatEntry(ctx, entry.SnatEntryId, entry.SnatTableId)
			waiter.Done(err)
			if err != nil {
				return err
			}
		}
		log.Info("deleting Eip Association used in SNatEntry for zone ...", "zoneName", zoneName)
		for ipAddress := range toUnAssociateEIPs {
			the_eip, err := c.actor.GetEIPByAddress(ctx, ipAddress)
			if err != nil {
				return err
			}
			if the_eip != nil && *the_eip.Status == "InUse" {
				log.Info("delete eip association", "eipId", the_eip.EipId)
				if err := c.actor.UnAssociateEIP(ctx, the_eip); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func (c *FlowContext) deleteEipAssociation(zoneName string) flow.TaskFn {
	return func(ctx context.Context) error {
		log := c.LogFromContext(ctx)
		log.Info("deleting Eip Association for zone ...", "zoneName", zoneName)
		child := c.getZoneChild(zoneName)
		eipId := child.Get(IdentifierZoneNATGWElasticIP)
		if eipId == nil {
			return nil
		}
		log.Info("delete eip association", "eipId", *eipId)
		eip, err := c.actor.GetEIP(ctx, *eipId)
		if err != nil {
			return err
		}
		if eip == nil {
			return nil
		}
		if *eip.Status == "InUse" {
			if err := c.actor.UnAssociateEIP(ctx, eip); err != nil {
				return err
			}
		}
		return nil
	}
}

func (c *FlowContext) deleteElasticIP(zoneName string) flow.TaskFn {
	return func(ctx context.Context) error {
		log := c.LogFromContext(ctx)
		log.Info("deleting Eip for zone ...", "zoneName", zoneName)
		child := c.getZoneChild(zoneName)
		if child.IsAlreadyDeleted(IdentifierZoneNATGWElasticIP) {
			return nil
		}
		zoneSuffix := c.getZoneSuffix(zoneName)
		eipSuffix := fmt.Sprintf("eip-natgw-%s", zoneSuffix)
		tags := c.commonTagsWithSuffix(eipSuffix)
		current, err := findExisting(ctx, child.Get(IdentifierZoneNATGWElasticIP), tags, c.actor.GetEIP, c.actor.FindEIPsByTags)
		if err != nil {
			return err
		}
		if current != nil {
			log.Info("deleting...", "AllocationId", current.EipId)
			waiter := informOnWaiting(log, 5*time.Second, "still deleting...", "AllocationId", current.EipId)
			err = c.actor.DeleteEIP(ctx, current.EipId)
			waiter.Done(err)
			if err != nil {
				return err
			}
		}
		child.SetAsDeleted(IdentifierZoneNATGWElasticIP)
		return nil
	}
}

func (c *FlowContext) deleteNatGatewayInVSwitches(vswitchIds []string) flow.TaskFn {
	return func(ctx context.Context) error {
		log := c.LogFromContext(ctx)
		log.Info("deleting managed natgateway in vswitches ...")
		current, err := findExisting(ctx, c.state.Get(IdentifierNatGateway), c.commonTagsWithSuffix("natgw"),
			c.actor.GetNatGateway, c.actor.FindNatGatewayByTags)
		if err != nil {
			return err
		}
		if current != nil && contains(vswitchIds, *current.VswitchId) {
			if err := c.deleteSNatEntryForNatGateway(ctx, current); err != nil {
				return err
			}

			if err := c.deleteNatGateway(ctx, current); err != nil {
				return err
			}
		}
		return nil
	}
}

func (c *FlowContext) deleteSNatEntryForNatGateway(ctx context.Context, ngw *aliclient.NatGateway) error {
	log := c.LogFromContext(ctx)
	log.Info("deleting snate entry for natgateway ...", "NatgatewayId", ngw.NatGatewayId)

	entries, err := c.actor.FindSNatEntriesByNatGateway(ctx, ngw.NatGatewayId)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		waiter := informOnWaiting(log, 5*time.Second, "still deleting snate entry ...", "SnatEntryId", entry.SnatEntryId, "SnatTableId", entry.SnatTableId)
		err := c.actor.DeleteSNatEntry(ctx, entry.SnatEntryId, entry.SnatTableId)
		waiter.Done(err)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *FlowContext) deleteNatGateway(ctx context.Context, ngw *aliclient.NatGateway) error {
	log := c.LogFromContext(ctx)
	log.Info("deleting natgateway ...", "NatgatewayId", ngw.NatGatewayId)
	waiter := informOnWaiting(log, 10*time.Second, "still deleting...", "NatGatewayID", ngw.NatGatewayId)
	err := c.actor.DeleteNatGateway(ctx, ngw.NatGatewayId)
	waiter.Done(err)
	if err != nil {
		return err
	}
	c.state.SetAsDeleted(IdentifierNatGateway)
	return nil
}

func (c *FlowContext) deleteVSwitch(vsw *aliclient.VSwitch) flow.TaskFn {
	return func(ctx context.Context) error {
		zoneChild := c.state.GetChild(ChildIdZones).GetChild(vsw.ZoneId)
		log := c.LogFromContext(ctx)
		log.Info("deleting vswitch ...", "VSwitchId", vsw.VSwitchId)

		if zoneChild.IsAlreadyDeleted(IdentifierZoneVSwitch) {
			return nil
		}
		if err := c.actor.DeleteVSwitch(ctx, vsw.VSwitchId); err != nil {
			return err
		}
		zoneChild.SetAsDeleted(IdentifierZoneVSwitch)

		return nil
	}
}

func (c *FlowContext) deleteZones(ctx context.Context) error {
	current, err := c.collectExistingVSwitches(ctx)
	if err != nil {
		return err
	}
	return c.DeleteZoneByVSwitches(ctx, current)
}

func (c *FlowContext) getZoneConfig(zoneName string) *alicloud.Zone {
	for _, zone := range c.config.Networks.Zones {
		if zone.Name == zoneName {
			return &zone
		}
	}
	return nil
}

func (c *FlowContext) getZoneChild(zoneName string) Whiteboard {
	return c.state.GetChild(ChildIdZones).GetChild(zoneName)
}
