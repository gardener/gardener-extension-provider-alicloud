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

	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
	"github.com/gardener/gardener/pkg/utils/flow"
)

// Delete creates and runs the flow to reconcile the Alicloud infrastructure.
func (c *FlowContext) Delete(ctx context.Context) error {
	if c.state.IsEmpty() {
		// nothing to do, e.g. if cluster was created with wrong credentials
		return nil
	}
	g := c.buildDeleteGraph()
	f := g.Compile()
	if err := f.Run(ctx, flow.Opts{Log: c.Log}); err != nil {
		return flow.Causes(err)
	}
	return nil
}

func (c *FlowContext) buildDeleteGraph() *flow.Graph {
	deleteVPC := c.config.Networks.VPC.ID == nil
	deleteNatGateway := deleteVPC || (c.config.Networks.VPC.GardenerManagedNATGateway != nil && *c.config.Networks.VPC.GardenerManagedNATGateway)

	g := flow.NewGraph("Alicloud infrastructure destruction")

	deleteNatgateway := c.AddTask(g, "delete natgateway",
		c.deleteNatGateway,
		DoIf(deleteNatGateway && c.hasNatGateway()), Timeout(defaultLongTimeout))

	deleteVSwitches := c.AddTask(g, "delete vswitch",
		c.deleteVSwitches,
		Timeout(defaultTimeout), Dependencies(deleteNatgateway))

	_ = c.AddTask(g, "delete VPC",
		c.deleteVpc,
		DoIf(deleteVPC && c.hasVPC()), Timeout(defaultTimeout), Dependencies(deleteVSwitches))

	return g
}

func (c *FlowContext) deleteVpc(ctx context.Context) error {
	if c.state.IsAlreadyDeleted(IdentifierVPC) {
		return nil
	}
	log := c.LogFromContext(ctx)
	current, err := findExisting(ctx, c.state.Get(IdentifierVPC), c.commonTags,
		c.actor.GetVpc, c.actor.FindVpcsByTags)
	if err != nil {
		return err
	}
	if current != nil {
		log.Info("deleting...", "VpcId", current.VpcId)
		if err := c.actor.DeleteVpc(ctx, current.VpcId); err != nil {
			return err
		}
	}
	c.state.SetAsDeleted(IdentifierVPC)
	return nil
}

func (c *FlowContext) deleteVSwitches(ctx context.Context) error {
	log := c.LogFromContext(ctx)
	current, err := c.collectExistingVSwitches(ctx)
	if err != nil {
		return err
	}
	for _, vsw := range current {
		log.Info("deleting...", "VSwitchId", vsw.VSwitchId)
		key := ChildIdZones + Separator + vsw.ZoneId + Separator + IdentifierZoneVSwitch
		if c.state.IsAlreadyDeleted(key) {
			continue
		}
		if err := c.actor.DeleteVSwitch(ctx, vsw.VSwitchId); err != nil {
			return err
		}
		c.state.SetAsDeleted(key)
	}
	return nil
}

func (c *FlowContext) deleteNatGateway(ctx context.Context) error {
	log := c.LogFromContext(ctx)
	current, err := findExisting(ctx, c.state.Get(IdentifierNatGateway), c.commonTags,
		c.actor.GetNatGateway, c.actor.FindNatGatewayByTags)
	if err != nil {
		return err
	}
	if current != nil {
		log.Info("deleting...", "NatgatewayId", current.NatGatewayId)
		if err := c.actor.DeleteNatGateway(ctx, current.NatGatewayId); err != nil {
			return err
		}
	}
	c.state.SetAsDeleted(IdentifierNatGateway)
	return nil
}
