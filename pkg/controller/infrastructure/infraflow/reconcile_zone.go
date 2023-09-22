// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package infraflow

import (
	"context"
	"fmt"
	"time"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
	"github.com/gardener/gardener/pkg/utils/flow"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
)

func (c *FlowContext) ensureZones(ctx context.Context) error {
	log := c.LogFromContext(ctx)
	log.Info("begin ensure zones")
	g := flow.NewGraph("Alicloud infrastructure : zones")

	for _, zone := range c.config.Networks.Zones {
		c.addZoneReconcileTasks(g, zone.Name)
	}

	f := g.Compile()
	if err := f.Run(ctx, flow.Opts{Log: c.Log}); err != nil {
		return flow.Causes(err)
	}
	return nil
}

func (c *FlowContext) addZoneReconcileTasks(g *flow.Graph, zoneName string) {
	ensureElasticIP := c.AddTask(g, "ensure elastic IP "+zoneName,
		c.ensureElasticIP(zoneName),
		Timeout(defaultLongTimeout))
	_ = c.AddTask(g, "ensure eip association "+zoneName,
		c.ensureEipAssociation(zoneName),
		Timeout(defaultLongTimeout), Dependencies(ensureElasticIP))
}

func (c *FlowContext) ensureEipAssociation(zoneName string) flow.TaskFn {
	return func(ctx context.Context) error {
		zone := c.getZoneConfig(zoneName)
		child := c.getZoneChild(zoneName)
		eipId := child.Get(IdentifierZoneNATGWElasticIP)
		ngwId := c.state.Get(IdentifierNatGateway)
		if eipId == nil && zone.NatGateway != nil && zone.NatGateway.EIPAllocationID != nil {
			eipId = zone.NatGateway.EIPAllocationID
		}
		if eipId == nil {
			return fmt.Errorf("no Eip exist @ zone %s", zoneName)
		}
		current, err := c.actor.GetEIP(ctx, *eipId)
		if err != nil {
			return err
		}
		if *current.Status == "Available" {
			// association
			err := c.actor.AssociateEIP(ctx, *eipId, *ngwId, "Nat")
			if err != nil {
				return err
			}
		} else if *current.Status == "InUse" {
			if *current.InstanceId != *ngwId {
				return fmt.Errorf("the eip %s is not associated to natgateway %s", *eipId, *ngwId)
			}

		} else {
			return fmt.Errorf(" eip %s status %s not allowed", *eipId, *current.Status)

		}
		return nil
	}
}

func (c *FlowContext) ensureElasticIP(zoneName string) flow.TaskFn {
	return func(ctx context.Context) error {
		zone := c.getZoneConfig(zoneName)
		if zone == nil {
			return fmt.Errorf("can not get zone config for %s", zoneName)
		}
		log := c.LogFromContext(ctx)
		if zone.NatGateway != nil && zone.NatGateway.EIPAllocationID != nil {

			eipId := *zone.NatGateway.EIPAllocationID
			log.Info("using configured EIP", "eipId", eipId)
			current, err := c.actor.GetEIP(ctx, eipId)
			if err != nil {
				return err
			}
			if current == nil {
				return fmt.Errorf("EIP %s has not been found", eipId)
			}
			return nil
		}

		zoneSuffix := c.getZoneSuffix(zone.Name)
		eipSuffix := fmt.Sprintf("eip-natgw-%s", zoneSuffix)
		child := c.getZoneChild(zone.Name)
		id := child.Get(IdentifierZoneNATGWElasticIP)
		desired := &aliclient.EIP{
			Name:               c.namespace + "-" + eipSuffix,
			Tags:               c.commonTagsWithSuffix(eipSuffix),
			Bandwidth:          "100",
			InternetChargeType: "PayByTraffic",
		}
		current, err := findExisting(ctx, id, desired.Tags, c.actor.GetEIP, c.actor.FindEIPsByTags)
		if err != nil {
			return err
		}

		if current != nil {
			child.Set(IdentifierZoneNATGWElasticIP, current.EipId)
			if _, err := c.updater.UpdateEIP(ctx, desired, current); err != nil {
				return err
			}
		} else {
			log.Info("creating...")
			created, err := c.actor.CreateEIP(ctx, desired)
			if err != nil {
				return err
			}
			child.Set(IdentifierZoneNATGWElasticIP, created.EipId)
			if _, err := c.updater.UpdateEIP(ctx, desired, created); err != nil {
				return err
			}
		}
		if err := c.PersistState(ctx, true); err != nil {
			return err
		}
		return nil
	}
}

func (c *FlowContext) ensureVSwitches(ctx context.Context) error {
	log := c.LogFromContext(ctx)
	var desired []*aliclient.VSwitch
	for _, zone := range c.config.Networks.Zones {
		zoneSuffix := c.getZoneSuffix(zone.Name)
		workerSuffix := fmt.Sprintf("nodes-%s", zoneSuffix)
		desired = append(desired,
			&aliclient.VSwitch{
				Name:      c.namespace + "-" + zone.Name + "-vsw",
				CidrBlock: zone.Workers,
				VpcId:     c.state.Get(IdentifierVPC),
				Tags:      c.commonTagsWithSuffix(workerSuffix),
				ZoneId:    zone.Name,
			})
	}

	current, err := c.collectExistingVSwitches(ctx)
	if err != nil {
		return err
	}

	toBeDeleted, toBeCreated, toBeChecked := diffByID(desired, current, func(item *aliclient.VSwitch) string {
		return item.ZoneId + "-" + item.CidrBlock
	})

	if err := c.DeleteZoneByVSwitches(ctx, toBeDeleted); err != nil {
		return err
	}

	for _, desired := range toBeCreated {
		log.Info("creating vswitch ...")
		created, err := c.actor.CreateVSwitch(ctx, desired)
		if err != nil {
			return err
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

func (c *FlowContext) DeleteZoneByVSwitches(ctx context.Context, toBeDeleted []*aliclient.VSwitch) error {

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

	deleteNatGateway := c.AddTask(g, "delete NatGateway",
		c.deleteNatGatewayInVSwitches(vswitchIds),
		DoIf(c.hasNatGateway()), Timeout(defaultLongTimeout), Dependencies(dependencies...))

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

	if err := c.PersistState(ctx, true); err != nil {
		return err
	}
	return nil
}

func (c *FlowContext) addZoneDeletionTasks(g *flow.Graph, zoneName string) flow.TaskIDer {

	deleteEipAssociation := c.AddTask(g, "delete eip association "+zoneName,
		c.deleteEipAssociation(zoneName),
		Timeout(defaultTimeout))

	deleteElasticIP := c.AddTask(g, "delete elastic IP "+zoneName,
		c.deleteElasticIP(zoneName),
		Timeout(defaultTimeout), Dependencies(deleteEipAssociation))

	return deleteElasticIP
}

func (c *FlowContext) deleteEipAssociation(zoneName string) flow.TaskFn {
	return func(ctx context.Context) error {
		zone := c.getZoneConfig(zoneName)
		child := c.getZoneChild(zoneName)
		eipId := child.Get(IdentifierZoneNATGWElasticIP)
		if eipId == nil && zone.NatGateway != nil && zone.NatGateway.EIPAllocationID != nil {
			eipId = zone.NatGateway.EIPAllocationID
		}
		if eipId == nil {
			return nil
		}
		current, err := c.actor.GetEIP(ctx, *eipId)
		if err != nil {
			return err
		}
		if *current.Status == "InUse" {
			if err := c.actor.UnAssociateEIP(ctx, current); err != nil {
				return err
			}
		}
		return nil
	}

}

func (c *FlowContext) deleteElasticIP(zoneName string) flow.TaskFn {
	return func(ctx context.Context) error {
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
			log := c.LogFromContext(ctx)
			log.Info("deleting...", "AllocationId", current.EipId)
			waiter := informOnWaiting(log, 10*time.Second, "still deleting...", "AllocationId", current.EipId)
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
		current, err := findExisting(ctx, c.state.Get(IdentifierNatGateway), c.commonTags,
			c.actor.GetNatGateway, c.actor.FindNatGatewayByTags)
		if err != nil {
			return err
		}
		if current != nil && contains(vswitchIds, *current.VswitchId) {
			log.Info("deleting natgateway ...", "NatgatewayId", current.NatGatewayId)
			waiter := informOnWaiting(log, 10*time.Second, "still deleting...", "NatGatewayID", current.NatGatewayId)
			err := c.actor.DeleteNatGateway(ctx, current.NatGatewayId)
			waiter.Done(err)
			if err != nil {
				return err
			}
			c.state.SetAsDeleted(IdentifierNatGateway)
		}
		return nil
	}
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
	if err := c.DeleteZoneByVSwitches(ctx, current); err != nil {
		return err
	}
	return nil
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
