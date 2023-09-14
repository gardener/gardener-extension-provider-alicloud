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
)

const (
	defaultTimeout     = 90 * time.Second
	defaultLongTimeout = 3 * time.Minute
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

	_ = c.AddTask(g, "ensure vswitch",
		c.ensureVSwitches,
		Timeout(defaultLongTimeout), Dependencies(ensureVpc))
	return g
}

func (c *FlowContext) ensureVpc(ctx context.Context) error {
	if c.config.Networks.VPC.ID != nil {
		return c.ensureExistingVpc(ctx)
	}
	return c.ensureManagedVpc(ctx)
}

func (c *FlowContext) ensureExistingVpc(ctx context.Context) error {
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
		log.Info("creating...")
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

	return nil
}

func (c *FlowContext) ensureVSwitches(ctx context.Context) error {
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

	if err := c.PersistState(ctx, true); err != nil {
		return err
	}
	current, err := c.collectExistingVSwitches(ctx)
	if err != nil {
		return err
	}

	toBeDeleted, toBeCreated, toBeChecked := diffByID(desired, current, func(item *aliclient.VSwitch) string {
		return item.ZoneId + "-" + item.CidrBlock
	})
	fmt.Println(toBeDeleted)
	fmt.Println(toBeCreated)
	fmt.Println(toBeChecked)
	for _, vsw := range toBeCreated {
		created, err := c.actor.CreateVSwitch(ctx, vsw)
		if err != nil {
			return err
		}
		c.state.GetChild(ChildIdZones).GetChild(vsw.ZoneId).Set(IdentifierZoneVSwitch, created.VSwitchId)
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
