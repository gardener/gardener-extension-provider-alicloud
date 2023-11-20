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

	"github.com/gardener/gardener/pkg/utils/flow"

	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
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
	g := flow.NewGraph("Alicloud infrastructure destruction")

	deleteZones := c.AddTask(g, "delete vswitch",
		c.deleteZones,
		Timeout(defaultTimeout))

	deleteSecurityGroup := c.AddTask(g, "delete security group",
		c.deleteSecurityGroup,
		Timeout(defaultTimeout))

	_ = c.AddTask(g, "delete VPC",
		c.deleteVpc,
		DoIf(deleteVPC && c.hasVPC()), Timeout(defaultTimeout), Dependencies(deleteZones, deleteSecurityGroup))

	return g
}

func (c *FlowContext) deleteSecurityGroup(ctx context.Context) error {
	if c.state.IsAlreadyDeleted(IdentifierNodesSecurityGroup) {
		return nil
	}
	log := c.LogFromContext(ctx)
	current, err := findExisting(ctx, c.state.Get(IdentifierNodesSecurityGroup), c.commonTagsWithSuffix("sg"),
		c.actor.GetSecurityGroup, c.actor.FindSecurityGroupsByTags)
	if err != nil {
		return err
	}
	if current != nil {
		log.Info("deleting security group ...", "GroupId", current.SecurityGroupId)
		for _, rule := range current.Rules {
			if err := c.actor.RevokeSecurityGroupRule(ctx, current.SecurityGroupId, rule.SecurityGroupRuleId, rule.Direction); err != nil {
				return err
			}
		}
		if err := c.actor.DeleteSecurityGroup(ctx, current.SecurityGroupId); err != nil {
			return err
		}
	}
	c.state.SetAsDeleted(IdentifierNodesSecurityGroup)
	return nil
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
		log.Info("deleting vpc ...", "VpcId", current.VpcId)
		if err := c.actor.DeleteVpc(ctx, current.VpcId); err != nil {
			return err
		}
	}
	c.state.SetAsDeleted(IdentifierVPC)
	return nil
}
