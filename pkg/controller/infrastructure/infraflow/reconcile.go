// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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

	g := flow.NewGraph("Alicloud infrastructure reconcilation")

	ensureVpc := c.AddTask(g, "ensure VPC",
		c.ensureVpc,
		Timeout(defaultTimeout))

	_ = c.AddTask(g, "ensure SecurityGroup",
		c.ensureSecurityGroup,
		Timeout(defaultLongTimeout), Dependencies(ensureVpc))

	ensureVSwitches := c.AddTask(g, "ensure vswitch",
		c.ensureVSwitches,
		Timeout(defaultLongTimeout), Dependencies(ensureVpc))

	ensureNatGateway := c.AddTask(g, "ensure natgateway",
		c.ensureNatGateway,
		Timeout(defaultLongTimeout), Dependencies(ensureVSwitches))

	_ = c.AddTask(g, "ensure zones",
		c.ensureZones,
		Timeout(defaultTimeout), Dependencies(ensureNatGateway))

	return g
}

func (c *FlowContext) ensureSecurityGroup(ctx context.Context) error {

	vpc, err := c.actor.GetVpc(ctx, *c.state.Get(IdentifierVPC))
	if err != nil {
		return err
	}
	log := c.LogFromContext(ctx)
	groupName := fmt.Sprintf("%s-sg", c.namespace)
	desired := &aliclient.SecurityGroup{
		Tags:        c.commonTagsWithSuffix("sg"),
		Name:        groupName,
		VpcId:       vpc.VpcId,
		Description: fmt.Sprintf("Security group for %s", c.namespace),
		Rules: []*aliclient.SecurityGroupRule{
			{
				Direction:    "ingress",
				Policy:       "Accept",
				Priority:     "1",
				IpProtocol:   "TCP",
				PortRange:    "30000/32767",
				SourceCidrIp: "0.0.0.0/0",
			},
			{
				Direction:    "ingress",
				Policy:       "Accept",
				Priority:     "1",
				IpProtocol:   "TCP",
				PortRange:    "1/22",
				SourceCidrIp: vpc.CidrBlock,
			},
			{
				Direction:    "ingress",
				Policy:       "Accept",
				Priority:     "1",
				IpProtocol:   "TCP",
				PortRange:    "24/513",
				SourceCidrIp: vpc.CidrBlock,
			},
			{
				Direction:    "ingress",
				Policy:       "Accept",
				Priority:     "1",
				IpProtocol:   "TCP",
				PortRange:    "515/65535",
				SourceCidrIp: vpc.CidrBlock,
			},
			{
				Direction:    "ingress",
				Policy:       "Accept",
				Priority:     "1",
				IpProtocol:   "UDP",
				PortRange:    "1/22",
				SourceCidrIp: vpc.CidrBlock,
			},
			{
				Direction:    "ingress",
				Policy:       "Accept",
				Priority:     "1",
				IpProtocol:   "UDP",
				PortRange:    "24/513",
				SourceCidrIp: vpc.CidrBlock,
			},
			{
				Direction:    "ingress",
				Policy:       "Accept",
				Priority:     "1",
				IpProtocol:   "UDP",
				PortRange:    "515/65535",
				SourceCidrIp: vpc.CidrBlock,
			},
		},
	}

	if c.cluster != nil {
		desired.Rules = append(desired.Rules, &aliclient.SecurityGroupRule{
			Direction:    "ingress",
			Policy:       "Accept",
			Priority:     "1",
			IpProtocol:   "ALL",
			PortRange:    "-1/-1",
			SourceCidrIp: *c.cluster.Shoot.Spec.Networking.Pods,
		})
	}
	current, err := findExisting(ctx, c.state.Get(IdentifierNodesSecurityGroup), c.commonTagsWithSuffix("sg"),
		c.actor.GetSecurityGroup, c.actor.FindSecurityGroupsByTags)
	if err != nil {
		return err
	}
	if current == nil {
		log.Info("creating security group ...")
		current, err = c.actor.CreateSecurityGroup(ctx, desired)
		if err != nil {
			return err
		}
	}
	c.state.Set(IdentifierNodesSecurityGroup, current.SecurityGroupId)
	if _, err := c.updater.UpdateSecurityGroup(ctx, desired, current); err != nil {
		return err
	}
	toBeDeleted, toBeCreated, _ := diffByID(desired.Rules, current.Rules, func(item *aliclient.SecurityGroupRule) string {
		return item.Direction + "-" + item.Policy + "-" + item.SourceCidrIp + "-" + item.DestCidrIp + "-" + item.PortRange + "-" + item.IpProtocol + "-" + item.Priority
	})
	for _, rule := range toBeDeleted {
		if err := c.actor.RevokeSecurityGroupRule(ctx, current.SecurityGroupId, rule.SecurityGroupRuleId, rule.Direction); err != nil {
			return err
		}
	}

	for _, rule := range toBeCreated {
		if err := c.actor.AuthorizeSecurityGroupRule(ctx, current.SecurityGroupId, *rule); err != nil {
			return err
		}
	}
	return c.PersistState(ctx, true)
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
	return c.PersistState(ctx, true)

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
	return c.PersistState(ctx, true)
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
		found, err := c.actor.ListVSwitches(ctx, ids)
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
	return c.PersistState(ctx, true)
}

func (c *FlowContext) ensureManagedNatGateway(ctx context.Context) error {

	log := c.LogFromContext(ctx)
	log.Info("using managed NatGateway")

	availableVSwitches := c.getAllVSwitchids()
	if len(availableVSwitches) == 0 {
		return fmt.Errorf("no available VSwitch can found for natgateway")
	}

	desired := &aliclient.NatGateway{
		Tags:               c.commonTagsWithSuffix("natgw"),
		Name:               c.namespace + "-natgw",
		VpcId:              c.state.Get(IdentifierVPC),
		AvailableVSwitches: availableVSwitches,
	}

	current, err := findExisting(ctx, c.state.Get(IdentifierNatGateway), c.commonTagsWithSuffix("natgw"),
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
		waiter := informOnWaiting(log, 10*time.Second, "still creating natgateway...")
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
	return c.PersistState(ctx, true)

}

func getZoneName(item *aliclient.VSwitch) string {
	return item.ZoneId
}
