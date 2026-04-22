// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
		Timeout(defaultLongTimeout))

	deleteSecurityGroup := c.AddTask(g, "delete security group",
		c.deleteSecurityGroup,
		Timeout(defaultTimeout))

	deleteRouteTable := c.AddTask(g, "delete route table",
		c.deleteRouteTable,
		DoIf(c.useCustomRouteTable()), Timeout(defaultTimeout), Dependencies(deleteZones))

	_ = c.AddTask(g, "delete VPC",
		c.deleteVpc,
		DoIf(deleteVPC && c.hasVPC()), Timeout(defaultTimeout), Dependencies(deleteZones, deleteSecurityGroup, deleteRouteTable))

	return g
}

func (c *FlowContext) deleteRouteTable(ctx context.Context) error {
	if c.state.IsAlreadyDeleted(IdentifierRouteTable) {
		return nil
	}
	log := c.LogFromContext(ctx)
	current, err := findExisting(ctx, c.state.Get(IdentifierRouteTable), c.commonTagsWithSuffix("rt"),
		c.actor.GetRouteTable, c.actor.FindRouteTablesByTags)
	if err != nil {
		return err
	}
	if current != nil {
		// 1. Unassociate any remaining vswitches (defensive cleanup after deleteZones)
		for _, vswId := range current.VSwitchIds {
			if err := c.actor.UnassociateRouteTable(ctx, current.RouteTableId, vswId); err != nil {
				return err
			}
		}

		// 2. Delete all custom route entries
		entries, err := c.actor.ListRouteEntriesByRouteTable(ctx, current.RouteTableId)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			log.Info("deleting route entry ...", "RouteEntryId", entry.RouteEntryId, "Dest", entry.DestinationCidrBlock)
			if err := c.actor.DeleteRouteEntry(ctx, current.RouteTableId, entry); err != nil {
				return err
			}
		}

		// 3. Delete the route table itself
		log.Info("deleting route table ...", "RouteTableId", current.RouteTableId)
		if err := c.actor.DeleteRouteTable(ctx, current.RouteTableId); err != nil {
			return err
		}
	}
	c.state.SetAsDeleted(IdentifierRouteTable)
	return c.PersistState(ctx, true)
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
