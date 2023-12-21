// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infraflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gardener/gardener/pkg/utils/flow"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/dualstack"
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

	_ = c.AddTask(g, "ensure DualStack",
		c.ensureDualStack,
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

func (c *FlowContext) ensureDualStack(ctx context.Context) error {
	enableDualStack := c.config.DualStack != nil && c.config.DualStack.Enabled
	if !enableDualStack {
		return nil
	}

	vpc, err := c.actor.GetVpc(ctx, *c.state.Get(IdentifierVPC))
	if err != nil {
		return err
	}
	if !vpc.EnableIpv6 {
		return fmt.Errorf("vpc is not enabled Ipv6")
	}
	dualStackValues, err := dualstack.CreateDualStackValues(true, c.infraSpec.Region, &vpc.CidrBlock, c.credentials)
	if err != nil {
		return err
	}

	if err = c.ensureIpv6Gateway(ctx); err != nil {
		return err
	}

	if err = c.ensureIpv6VSwitches(ctx, dualStackValues.Zone_A_CIDR, dualStackValues.Zone_A, IdentifierDualStackVSwitch_A, dualStackValues.Zone_A_IPV6_MASK); err != nil {
		return err
	}

	err = c.ensureIpv6VSwitches(ctx, dualStackValues.Zone_B_CIDR, dualStackValues.Zone_B, IdentifierDualStackVSwitch_B, dualStackValues.Zone_B_IPV6_MASK)
	return err
}

func (c *FlowContext) ensureIpv6Gateway(ctx context.Context) error {
	log := c.LogFromContext(ctx)
	log.Info("ensureIpv6Gateway")
	suffix := "DUAL_STACK-ipv6-gw"
	desired := &aliclient.IPV6Gateway{
		Tags:  c.commonTagsWithSuffix(suffix),
		Name:  c.namespace + "-" + suffix,
		VpcId: *c.state.Get(IdentifierVPC),
	}

	current, err := findExisting(ctx, c.state.Get(IdentifierIPV6Gateway), c.commonTagsWithSuffix(suffix),
		c.actor.GetIpv6Gateway, c.actor.FindIpv6GatewaysByTags)

	if err != nil {
		return err
	}
	if current != nil {
		c.state.Set(IdentifierIPV6Gateway, current.IPV6GatewayId)

		_, err := c.updater.UpdateIpv6Gateway(ctx, desired, current)
		if err != nil {
			return err
		}
	} else {
		log.Info("creating ipv6gateway ...")
		waiter := informOnWaiting(log, 10*time.Second, "still creating ipv6gateway...")
		created, err := c.actor.CreateIpv6Gateway(ctx, desired)
		waiter.Done(err)
		if err != nil {
			return fmt.Errorf("create Ipv6Gateway failed %w", err)
		}

		c.state.Set(IdentifierIPV6Gateway, created.IPV6GatewayId)
		_, err = c.updater.UpdateIpv6Gateway(ctx, desired, created)
		if err != nil {
			return err
		}

	}
	return c.PersistState(ctx, true)
}

func (c *FlowContext) ensureIpv6VSwitches(ctx context.Context, cidrBlock, zoneId, vswitchIdentifier string, ipv6CidrMask int) error {
	log := c.LogFromContext(ctx)
	log.Info("ensureIpv6VSwitches:" + vswitchIdentifier)
	suffix := vswitchIdentifier
	desired := &aliclient.VSwitch{
		Name:          c.namespace + "-" + suffix,
		CidrBlock:     cidrBlock,
		VpcId:         c.state.Get(IdentifierVPC),
		Tags:          c.commonTagsWithSuffix(suffix),
		ZoneId:        zoneId,
		EnableIpv6:    true,
		Ipv6CidrkMask: &ipv6CidrMask,
	}

	current, err := findExisting(ctx, c.state.Get(vswitchIdentifier), c.commonTagsWithSuffix(suffix),
		c.actor.GetVSwitch, c.actor.FindVSwitchesByTags)

	if err != nil {
		return err
	}
	if current != nil {
		c.state.Set(vswitchIdentifier, current.VSwitchId)

		_, err := c.updater.UpdateVSwitch(ctx, desired, current)
		if err != nil {
			return err
		}
	} else {
		log.Info("creating ipv6 vswitch ...")
		waiter := informOnWaiting(log, 10*time.Second, "still creating ipv6 vswitch...")
		created, err := c.actor.CreateVSwitch(ctx, desired)
		waiter.Done(err)
		if err != nil {
			return fmt.Errorf("create pv6 vswitch failed %w", err)
		}

		c.state.Set(vswitchIdentifier, created.VSwitchId)
		_, err = c.updater.UpdateVSwitch(ctx, desired, created)
		if err != nil {
			return err
		}

	}
	return c.PersistState(ctx, true)

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
		return fmt.Errorf("vpc %s has not been found", vpcID)
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
	enableDualStack := c.config.DualStack != nil && c.config.DualStack.Enabled
	desired := &aliclient.VPC{
		Tags:       c.commonTags,
		CidrBlock:  *c.config.Networks.VPC.CIDR,
		Name:       c.namespace + "-vpc",
		EnableIpv6: enableDualStack,
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
		if tagName, ok := item.Tags["Name"]; ok && strings.Contains(tagName, "nodes-z") {
			for _, currentItem := range current {
				if item.VSwitchId == currentItem.VSwitchId {
					continue outer
				}
			}
			current = append(current, item)
		}
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
