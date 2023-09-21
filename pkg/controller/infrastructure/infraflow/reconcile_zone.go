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
)

func (c *FlowContext) reconcileZones(ctx context.Context) error {
	log := c.LogFromContext(ctx)
	var desired []*aliclient.VSwitch
	for _, zone := range c.config.Networks.Zones {
		zoneSuffix := c.getZoneSuffix(zone.Name)
		workerSuffix := fmt.Sprintf("nodes-%s", zoneSuffix)
		desired = append(desired,
			&aliclient.VSwitch{
				Name:      c.namespace + zone.Name + "-vsw",
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
	return nil
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
