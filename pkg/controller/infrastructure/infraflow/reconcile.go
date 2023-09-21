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

	"github.com/gardener/gardener/pkg/utils/flow"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
)

const (
	defaultTimeout     = 90 * time.Second
	defaultLongTimeout = 5 * time.Minute
)

// Reconcile creates and runs the flow to reconcile the Alicloud infrastructure.
func (c *FlowContext) Reconcile(ctx context.Context) error {
	g := c.buildReconcileGraph()
	f := g.Compile()
	if err := f.Run(ctx, flow.Opts{Log: c.Log}); err != nil {
		return flow.Causes(err)
	}
	return nil
}

func (c *FlowContext) buildReconcileGraph() *flow.Graph {
	createVPC := c.config.Networks.VPC.ID == nil
	createNatGateway := createVPC || (c.config.Networks.VPC.GardenerManagedNATGateway != nil && *c.config.Networks.VPC.GardenerManagedNATGateway)
	fmt.Println(createNatGateway)

	g := flow.NewGraph("Alicloud infrastructure reconcilation")

	ensureVpc := c.AddTask(g, "ensure VPC",
		c.ensureVpc,
		Timeout(defaultTimeout))

	ensureVSwitches := c.AddTask(g, "ensure vswitch",
		c.reconcileZones,
		Timeout(defaultLongTimeout), Dependencies(ensureVpc))

	_ = c.AddTask(g, "ensure natgateway",
		c.ensureNatGateway,
		Timeout(defaultLongTimeout), Dependencies(ensureVSwitches))

	return g
}

func (c *FlowContext) ensureVpc(ctx context.Context) error {
	if c.config.Networks.VPC.ID != nil {
		return c.ensureExistingVpc(ctx)
	}
	return c.ensureManagedVpc(ctx)
}

func (c *FlowContext) ensureExistingVpc(ctx context.Context) error {
	vpcID := *c.config.Networks.VPC.ID
	log := c.LogFromContext(ctx)
	log.Info("using configured VPC", "vpc", vpcID)
	current, err := c.actor.GetVpc(ctx, vpcID)
	if err != nil {
		return err
	}
	if current == nil {
		return fmt.Errorf("VPC %s has not been found", vpcID)
	}
	c.state.Set(IdentifierVPC, vpcID)
	if err := c.PersistState(ctx, true); err != nil {
		return err
	}
	return nil

}

func (c *FlowContext) ensureManagedVpc(ctx context.Context) error {
	log := c.LogFromContext(ctx)
	log.Info("using managed VPC")

	if c.config.Networks.VPC.CIDR == nil {
		return fmt.Errorf("missing VPC CIDR")
	}

	desired := &aliclient.VPC{
		Tags:      c.commonTags,
		CidrBlock: *c.config.Networks.VPC.CIDR,
		Name:      c.namespace + "-vpc",
	}

	current, err := findExisting(ctx, c.state.Get(IdentifierVPC), c.commonTags,
		c.actor.GetVpc, c.actor.FindVpcsByTags)

	if err != nil {
		return err
	}
	if current != nil {
		c.state.Set(IdentifierVPC, current.VpcId)

		_, err := c.updater.UpdateVpc(ctx, desired, current)
		if err != nil {
			return err
		}
	} else {
		log.Info("creating vpc ...")
		created, err := c.actor.CreateVpc(ctx, desired)
		if err != nil {
			return fmt.Errorf("create VPC failed %w", err)
		}

		c.state.Set(IdentifierVPC, created.VpcId)
		_, err = c.updater.UpdateVpc(ctx, desired, created)
		if err != nil {
			return err
		}

	}
	if err := c.PersistState(ctx, true); err != nil {
		return err
	}

	return nil
}

func (c *FlowContext) collectExistingVSwitches(ctx context.Context) ([]*aliclient.VSwitch, error) {
	child := c.state.GetChild(ChildIdZones)
	var ids []string
	for _, zoneKey := range child.GetChildrenKeys() {
		zoneChild := child.GetChild(zoneKey)
		if id := zoneChild.Get(IdentifierZoneVSwitch); id != nil {
			ids = append(ids, *id)
		}
	}
	var current []*aliclient.VSwitch
	if len(ids) > 0 {
		found, err := c.actor.GetVSwitches(ctx, ids)
		if err != nil {
			return nil, err
		}
		current = found
	}
	foundByTags, err := c.actor.FindVSwitchesByTags(ctx, c.clusterTags())
	if err != nil {
		return nil, err
	}
outer:
	for _, item := range foundByTags {
		for _, currentItem := range current {
			if item.VSwitchId == currentItem.VSwitchId {
				continue outer
			}
		}
		current = append(current, item)
	}
	return current, nil
}

func (c *FlowContext) ensureNatGateway(ctx context.Context) error {

	createNatGateway := c.config.Networks.VPC.ID == nil || (c.config.Networks.VPC.GardenerManagedNATGateway != nil && *c.config.Networks.VPC.GardenerManagedNATGateway)

	if !createNatGateway {
		return c.ensureExistingNatGateway(ctx)
	}
	return c.ensureManagedNatGateway(ctx)
}
func (c *FlowContext) ensureExistingNatGateway(ctx context.Context) error {
	vpcID := c.state.Get(IdentifierVPC)
	gw, err := c.actor.FindNatGatewayByVPC(ctx, *vpcID)
	if err != nil {
		return fmt.Errorf("find NatGateway failed %w", err)
	}
	c.state.Set(IdentifierNatGateway, gw.NatGatewayId)
	if err := c.PersistState(ctx, true); err != nil {
		return err
	}
	return nil
}

func (c *FlowContext) ensureManagedNatGateway(ctx context.Context) error {

	log := c.LogFromContext(ctx)
	log.Info("using managed NatGateway")

	availableVSwitches := c.getAllVSwitchid()
	if len(availableVSwitches) == 0 {
		return fmt.Errorf("no available VSwitchcan found for natgateway")
	}

	desired := &aliclient.NatGateway{
		Tags:               c.commonTagsWithSuffix("natgw"),
		Name:               c.namespace + "-natgw",
		VpcId:              c.state.Get(IdentifierVPC),
		AvailableVSwitches: availableVSwitches,
	}

	current, err := findExisting(ctx, c.state.Get(IdentifierNatGateway), c.commonTags,
		c.actor.GetNatGateway, c.actor.FindNatGatewayByTags)

	if err != nil {
		return err
	}
	if current != nil {
		c.state.Set(IdentifierNatGateway, current.NatGatewayId)
		if !contains(desired.AvailableVSwitches, *current.VswitchId) {
			return fmt.Errorf("the natgateway should be deleted")
		}
		_, err := c.updater.UpdateNatgateway(ctx, desired, current)
		if err != nil {
			return err
		}
	} else {
		log.Info("creating natgateway ...")
		waiter := informOnWaiting(log, 10*time.Second, "still createing natgateway...")
		created, err := c.actor.CreateNatGateway(ctx, desired)
		waiter.Done(err)
		if err != nil {
			return fmt.Errorf("create NatGateway failed %w", err)
		}

		c.state.Set(IdentifierNatGateway, created.NatGatewayId)
		_, err = c.updater.UpdateNatgateway(ctx, desired, created)
		if err != nil {
			return err
		}

	}
	if err := c.PersistState(ctx, true); err != nil {
		return err
	}
	return nil

}

func getZoneName(item *aliclient.VSwitch) string {
	return item.ZoneId
}
