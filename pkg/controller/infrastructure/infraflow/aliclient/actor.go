// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aliclient

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	alierrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/log"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
)

// Actor ia a interface to package alicloud api call
type Actor interface {
	ListEnhanhcedNatGatewayAvailableZones(ctx context.Context, region string) ([]string, error)

	CreateVpc(ctx context.Context, vpc *VPC) (*VPC, error)
	GetVpc(ctx context.Context, id string) (*VPC, error)
	ListVpcs(ctx context.Context, ids []string) ([]*VPC, error)
	FindVpcsByTags(ctx context.Context, tags Tags) ([]*VPC, error)
	DeleteVpc(ctx context.Context, id string) error

	CreateVSwitch(ctx context.Context, vsw *VSwitch) (*VSwitch, error)
	GetVSwitch(ctx context.Context, id string) (*VSwitch, error)
	ListVSwitches(ctx context.Context, ids []string) ([]*VSwitch, error)
	FindVSwitchesByTags(ctx context.Context, tags Tags) ([]*VSwitch, error)
	FindVSwitchesByVPC(ctx context.Context, vpcId string) ([]*VSwitch, error)
	DeleteVSwitch(ctx context.Context, id string) error

	CreateNatGateway(ctx context.Context, ngw *NatGateway) (*NatGateway, error)
	GetNatGateway(ctx context.Context, id string) (*NatGateway, error)
	ListNatGateways(ctx context.Context, ids []string) ([]*NatGateway, error)
	FindNatGatewayByTags(ctx context.Context, tags Tags) ([]*NatGateway, error)
	FindNatGatewayByVPC(ctx context.Context, vpcId string) (*NatGateway, error)
	DeleteNatGateway(ctx context.Context, id string) error
	ListNatGatewaysByVSwitchInVPC(ctx context.Context, vpcId string, vswitchIds []string) ([]*NatGateway, error)

	CreateEIP(ctx context.Context, eip *EIP) (*EIP, error)
	GetEIP(ctx context.Context, id string) (*EIP, error)
	GetEIPByAddress(ctx context.Context, ipAddress string) (*EIP, error)
	ListEIPs(ctx context.Context, ids []string) ([]*EIP, error)
	FindEIPsByTags(ctx context.Context, tags Tags) ([]*EIP, error)
	DeleteEIP(ctx context.Context, id string) error
	ModifyEIP(ctx context.Context, id string, eip *EIP) error
	AssociateEIP(ctx context.Context, id, to, insType string) error
	UnAssociateEIP(ctx context.Context, eip *EIP) error

	CreateSNatEntry(ctx context.Context, entry *SNATEntry) (*SNATEntry, error)
	GetSNatEntry(ctx context.Context, id, snatTableId string) (*SNATEntry, error)
	FindSNatEntriesByNatGateway(ctx context.Context, ngwId string) ([]*SNATEntry, error)
	DeleteSNatEntry(ctx context.Context, id, snatTableId string) error

	CreateTags(ctx context.Context, resources []string, tags Tags, resourceType string) error
	DeleteTags(ctx context.Context, resources []string, tags Tags, resourceType string) error

	CreateSecurityGroup(ctx context.Context, sg *SecurityGroup) (*SecurityGroup, error)
	GetSecurityGroup(ctx context.Context, id string) (*SecurityGroup, error)
	ListSecurityGroups(ctx context.Context, ids []string) ([]*SecurityGroup, error)
	FindSecurityGroupsByTags(ctx context.Context, tags Tags) ([]*SecurityGroup, error)
	DeleteSecurityGroup(ctx context.Context, id string) error

	AuthorizeSecurityGroupRule(ctx context.Context, sgId string, rule SecurityGroupRule) error
	RevokeSecurityGroupRule(ctx context.Context, sgId, ruleId, direction string) error
}

type actor struct {
	vpcClient    alicloudclient.VPC
	ecsClient    alicloudclient.ECS
	Logger       logr.Logger
	PollInterval time.Duration
}

var _ Actor = &actor{}

// NewActor is to create a Actor object
func NewActor(accessKeyID, secretAccessKey, region string) (Actor, error) {
	clientFactory := alicloudclient.NewClientFactory()
	vpcClient, err := clientFactory.NewVPCClient(region, accessKeyID, secretAccessKey)
	if err != nil {
		return nil, err
	}
	ecsClient, err := clientFactory.NewECSClient(region, accessKeyID, secretAccessKey)
	if err != nil {
		return nil, err
	}
	return &actor{
		vpcClient:    vpcClient,
		ecsClient:    ecsClient,
		Logger:       log.Log.WithName("alicloud-client"),
		PollInterval: 5 * time.Second,
	}, nil
}

func (c *actor) CreateTags(_ context.Context, resources []string, tags Tags, resourceType string) error {
	typeClass := c.getResourceClass(resourceType)
	switch typeClass {
	case "vpc":
		return c.createVpcTags(resources, tags, resourceType)
	case "ecs":
		return c.createEcsTags(resources, tags, resourceType)
	}
	return fmt.Errorf("unknown resource type %s", resourceType)
}

func (c *actor) DeleteTags(_ context.Context, resources []string, tags Tags, resourceType string) error {
	typeClass := c.getResourceClass(resourceType)
	switch typeClass {
	case "vpc":
		return c.deleteVpcTags(resources, tags, resourceType)
	case "ecs":
		return c.deleteEcsTags(resources, tags, resourceType)
	}
	return fmt.Errorf("unknown resource type %s", resourceType)
}

func (c *actor) getResourceClass(resourceType string) string {
	vpc_resourceType_list := []string{
		"VPC",
		"VSWITCH",
		"ROUTETABLE",
		"EIP",
		"VpnGateWay",
		"NATGATEWAY",
		"COMMONBANDWIDTHPACKAGE",
	}
	ecs_resourceType_list := []string{
		"instance",
		"disk",
		"snapshot",
		"image",
		"securitygroup",
		"volume",
		"eni",
		"ddh",
		"ddhcluster",
		"keypair",
		"launchtemplate",
		"reservedinstance",
		"snapshotpolicy",
		"elasticityassurance",
		"capacityreservation",
		"command",
		"invocation",
		"activation",
		"managedinstance",
	}
	if contains(vpc_resourceType_list, resourceType) {
		return "vpc"
	}
	if contains(ecs_resourceType_list, resourceType) {
		return "ecs"
	}
	return "unknown"
}

func (c *actor) ListEnhanhcedNatGatewayAvailableZones(_ context.Context, region string) ([]string, error) {
	req := vpc.CreateListEnhanhcedNatGatewayAvailableZonesRequest()
	req.RegionId = region

	resp, err := callApi(c.vpcClient.ListEnhanhcedNatGatewayAvailableZones, req)

	if err != nil {
		return nil, err
	}
	zoneIDs := make([]string, 0, len(resp.Zones))
	for _, zone := range resp.Zones {
		zoneIDs = append(zoneIDs, zone.ZoneId)
	}
	return zoneIDs, nil
}

func (c *actor) CreateSecurityGroup(ctx context.Context, sg *SecurityGroup) (*SecurityGroup, error) {
	req := ecs.CreateCreateSecurityGroupRequest()
	req.SecurityGroupName = sg.Name
	req.VpcId = sg.VpcId
	req.Description = sg.Description

	resp, err := callApi(c.ecsClient.CreateSecurityGroup, req)
	if err != nil {
		return nil, err
	}
	return c.GetSecurityGroup(ctx, resp.SecurityGroupId)
}

func (c *actor) GetSecurityGroup(_ context.Context, id string) (*SecurityGroup, error) {
	return c.getSecurityGroup(id)
}

func (c *actor) getSecurityGroup(id string) (*SecurityGroup, error) {
	req := ecs.CreateDescribeSecurityGroupsRequest()
	req.SecurityGroupId = id
	resp, err := c.describeSecurityGroup(req)

	sg, err := single(resp, err)
	if err != nil {
		return nil, err
	}
	if sg != nil {
		rules, err := c.listSecurityGroupRule(sg.SecurityGroupId)
		if err != nil {
			return nil, err
		}

		sg.Rules = append(sg.Rules, rules...)
	}
	return sg, nil
}

func (c *actor) ListSecurityGroups(_ context.Context, ids []string) ([]*SecurityGroup, error) {
	return listByIds(c.getSecurityGroup, ids)
}

func (c *actor) FindSecurityGroupsByTags(ctx context.Context, tags Tags) ([]*SecurityGroup, error) {
	req := ecs.CreateListTagResourcesRequest()
	req.ResourceType = "securitygroup"

	var reqTag []ecs.ListTagResourcesTag
	for k, v := range tags {
		reqTag = append(reqTag, ecs.ListTagResourcesTag{Key: k, Value: v})
	}
	req.Tag = &reqTag

	idList, err := c.listEcsTagResources(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.ListSecurityGroups(ctx, idList)
}
func (c *actor) DeleteSecurityGroup(ctx context.Context, id string) error {
	sg, err := c.getSecurityGroup(id)
	if err != nil {
		return err
	}
	if sg == nil {
		return nil
	}
	for _, rule := range sg.Rules {
		if err := c.RevokeSecurityGroupRule(ctx, id, rule.SecurityGroupRuleId, rule.Direction); err != nil {
			return err
		}
	}

	req := ecs.CreateDeleteSecurityGroupRequest()
	req.SecurityGroupId = id
	_, err = callApi(c.ecsClient.DeleteSecurityGroup, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *actor) listSecurityGroupRule(sgId string) ([]*SecurityGroupRule, error) {
	var rule_list []*SecurityGroupRule
	req := ecs.CreateDescribeSecurityGroupAttributeRequest()
	req.SecurityGroupId = sgId

	resp, err := callApi(c.ecsClient.DescribeSecurityGroupAttribute, req)
	if err != nil {
		return nil, err
	}

	for _, permission := range resp.Permissions.Permission {
		the_rule, err := c.fromSecurityGroupRule(permission)
		if err != nil {
			return nil, err
		}
		rule_list = append(rule_list, the_rule)
	}
	return rule_list, nil
}

func (c *actor) AuthorizeSecurityGroupRule(_ context.Context, sgId string, rule SecurityGroupRule) error {
	switch rule.Direction {
	case "ingress":
		return c.addIngressSecurityGroupRule(sgId, rule)
	case "egress":
		return c.addEgressSecurityGroupRule(sgId, rule)
	}
	return nil
}

func (c *actor) addIngressSecurityGroupRule(sgId string, rule SecurityGroupRule) error {
	req := ecs.CreateAuthorizeSecurityGroupRequest()
	req.SecurityGroupId = sgId
	req.Permissions = &[]ecs.AuthorizeSecurityGroupPermissions{
		{
			Policy:       rule.Policy,
			Priority:     rule.Priority,
			IpProtocol:   rule.IpProtocol,
			SourceCidrIp: rule.SourceCidrIp,
			PortRange:    rule.PortRange,
		},
	}

	_, err := callApi(c.ecsClient.AuthorizeSecurityGroup, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *actor) addEgressSecurityGroupRule(sgId string, rule SecurityGroupRule) error {
	req := ecs.CreateAuthorizeSecurityGroupEgressRequest()
	req.SecurityGroupId = sgId
	req.Permissions = &[]ecs.AuthorizeSecurityGroupEgressPermissions{
		{
			Policy:     rule.Policy,
			Priority:   rule.Priority,
			IpProtocol: rule.IpProtocol,
			PortRange:  rule.PortRange,
			DestCidrIp: rule.DestCidrIp,
		},
	}

	_, err := callApi(c.ecsClient.AuthorizeSecurityGroupEgress, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *actor) RevokeSecurityGroupRule(_ context.Context, sgId, ruleId, direction string) error {
	switch direction {
	case "ingress":
		return c.removeIngressSecurityGroupRule(sgId, ruleId)
	case "egress":
		return c.removeEgressSecurityGroupRule(sgId, ruleId)
	}
	return nil
}

func (c *actor) removeIngressSecurityGroupRule(sgId, ruleId string) error {
	req := ecs.CreateRevokeSecurityGroupRequest()
	req.SecurityGroupId = sgId
	req.SecurityGroupRuleId = &[]string{
		ruleId,
	}
	_, err := callApi(c.ecsClient.RevokeSecurityGroup, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *actor) removeEgressSecurityGroupRule(sgId, ruleId string) error {
	req := ecs.CreateRevokeSecurityGroupEgressRequest()
	req.SecurityGroupId = sgId
	req.SecurityGroupRuleId = &[]string{
		ruleId,
	}
	_, err := callApi(c.ecsClient.RevokeSecurityGroupEgress, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *actor) createEcsTags(resources []string, tags Tags, resourceType string) error {
	req := ecs.CreateTagResourcesRequest()
	req.ResourceType = resourceType
	req.ResourceId = &resources

	var reqTag []ecs.TagResourcesTag
	for k, v := range tags {
		reqTag = append(reqTag, ecs.TagResourcesTag{Key: k, Value: v})
	}
	req.Tag = &reqTag

	_, err := callApi(c.ecsClient.TagResources, req)
	return err
}

func (c *actor) deleteEcsTags(resources []string, tags Tags, resourceType string) error {
	req := ecs.CreateUntagResourcesRequest()
	req.ResourceType = resourceType
	req.ResourceId = &resources

	var reqTag []string
	for k := range tags {
		reqTag = append(reqTag, k)
	}
	req.TagKey = &reqTag
	_, err := callApi(c.ecsClient.UntagResources, req)
	return err
}

func (c *actor) FindSNatEntriesByNatGateway(_ context.Context, ngwId string) ([]*SNATEntry, error) {
	req := vpc.CreateDescribeSnatTableEntriesRequest()
	req.NatGatewayId = ngwId

	resp, err := c.describeSNATEntry(req)
	if err != nil {
		if serverErr, ok := err.(*alierrors.ServerError); ok {
			if serverErr.ErrorCode() == "InvalidSnatTableId.NotFound" {
				var entryList []*SNATEntry
				return entryList, nil
			}
		}
		return nil, err
	}
	return resp, nil
}

func (c *actor) CreateSNatEntry(ctx context.Context, entry *SNATEntry) (*SNATEntry, error) {
	req := vpc.CreateCreateSnatEntryRequest()
	req.SnatTableId = entry.SnatTableId
	req.SourceVSwitchId = entry.VSwitchId
	req.SnatIp = entry.IpAddress
	req.SnatEntryName = entry.Name

	resp, err := callApi(c.vpcClient.CreateSnatEntry, req)
	if err != nil {
		return nil, err
	}

	var created *SNATEntry
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		created, err = c.GetSNatEntry(ctx, resp.SnatEntryId, entry.SnatTableId)
		if err != nil {
			return false, err
		}
		if created == nil {
			return false, nil
		}
		if *created.Status != "Available" {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return created, nil
}
func (c *actor) GetSNatEntry(_ context.Context, id, snatTableId string) (*SNATEntry, error) {
	return c.getSNatEntry(id, snatTableId)
}

func (c *actor) getSNatEntry(id, snatTableId string) (*SNATEntry, error) {
	req := vpc.CreateDescribeSnatTableEntriesRequest()
	req.SnatEntryId = id
	req.SnatTableId = snatTableId
	resp, err := c.describeSNATEntry(req)

	return single(resp, err)
}

func (c *actor) DeleteSNatEntry(ctx context.Context, id, snatTableId string) error {
	current, err := c.GetSNatEntry(ctx, id, snatTableId)
	if err != nil {
		return err
	}
	if current == nil {
		return nil
	}
	req := vpc.CreateDeleteSnatEntryRequest()
	req.SnatEntryId = id
	req.SnatTableId = snatTableId
	_, err = callApi(c.vpcClient.DeleteSnatEntry, req)
	if err != nil {
		return err
	}
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		current, err := c.GetSNatEntry(ctx, id, snatTableId)
		if err != nil {
			return false, err
		}
		if current == nil {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return err
	}
	return nil
}

func (c *actor) ModifyEIP(_ context.Context, id string, eip *EIP) error {
	req := vpc.CreateModifyEipAddressAttributeRequest()
	req.AllocationId = id
	req.Bandwidth = eip.Bandwidth
	_, err := callApi(c.vpcClient.ModifyEipAddressAttribute, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *actor) AssociateEIP(ctx context.Context, id, to, insType string) error {
	req := vpc.CreateAssociateEipAddressRequest()
	req.AllocationId = id
	req.InstanceId = to
	req.InstanceType = insType
	_, err := callApi(c.vpcClient.AssociateEipAddress, req)
	if err != nil {
		return err
	}

	var theEip *EIP
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		theEip, err = c.GetEIP(ctx, id)

		if err != nil {
			return false, err
		}
		if *theEip.Status != "InUse" {
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return err
	}
	if *theEip.InstanceId != to {
		return fmt.Errorf("the eip %s is not associated to the target %s", id, to)
	}

	return nil
}

func (c *actor) UnAssociateEIP(ctx context.Context, eip *EIP) error {
	req := vpc.CreateUnassociateEipAddressRequest()
	req.AllocationId = eip.EipId
	req.InstanceId = *eip.InstanceId
	req.InstanceType = *eip.InstanceType

	_, err := callApi(c.vpcClient.UnassociateEipAddress, req)
	if err != nil {
		return err
	}

	var theEip *EIP
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		theEip, err = c.GetEIP(ctx, eip.EipId)

		if err != nil {
			return false, err
		}
		if *theEip.Status != "Available" {
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *actor) CreateEIP(ctx context.Context, eip *EIP) (*EIP, error) {
	req := vpc.CreateAllocateEipAddressRequest()
	req.Name = eip.Name
	req.Bandwidth = eip.Bandwidth
	req.InstanceChargeType = "PostPaid"
	req.InternetChargeType = eip.InternetChargeType

	resp, err := callApi(c.vpcClient.AllocateEipAddress, req)
	if err != nil {
		return nil, err
	}

	var created *EIP
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		created, err = c.GetEIP(ctx, resp.AllocationId)
		if err != nil {
			return false, err
		}
		if created == nil {
			return false, nil
		}
		if *created.Status != "Available" {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return created, nil
}

func (c *actor) GetEIPByAddress(_ context.Context, ipAddress string) (*EIP, error) {
	req := vpc.CreateDescribeEipAddressesRequest()
	req.EipAddress = ipAddress
	resp, err := c.describeEIP(req)

	return single(resp, err)
}

func (c *actor) GetEIP(_ context.Context, id string) (*EIP, error) {
	return c.getEIP(id)
}

func (c *actor) getEIP(id string) (*EIP, error) {
	req := vpc.CreateDescribeEipAddressesRequest()
	req.AllocationId = id
	resp, err := c.describeEIP(req)

	return single(resp, err)
}

func (c *actor) ListEIPs(_ context.Context, ids []string) ([]*EIP, error) {
	return listByIds(c.getEIP, ids)
}

func (c *actor) FindEIPsByTags(ctx context.Context, tags Tags) ([]*EIP, error) {
	req := vpc.CreateListTagResourcesRequest()
	req.ResourceType = "EIP"

	var reqTag []vpc.ListTagResourcesTag
	for k, v := range tags {
		reqTag = append(reqTag, vpc.ListTagResourcesTag{Key: k, Value: v})
	}
	req.Tag = &reqTag

	idList, err := c.listVpcTagResources(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.ListEIPs(ctx, idList)
}

func (c *actor) DeleteEIP(ctx context.Context, id string) error {
	current, err := c.GetEIP(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return nil
	}
	req := vpc.CreateReleaseEipAddressRequest()
	req.AllocationId = id
	_, err = callApi(c.vpcClient.ReleaseEipAddress, req)
	if err != nil {
		return err
	}
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		current, err := c.GetEIP(ctx, id)
		if err != nil {
			return false, err
		}
		if current == nil {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return err
	}
	return nil
}

func (c *actor) CreateNatGateway(ctx context.Context, ngw *NatGateway) (*NatGateway, error) {
	if len(ngw.AvailableVSwitches) == 0 {
		return nil, fmt.Errorf("length of AvailableVSwitches is 0")
	}

	req := vpc.CreateCreateNatGatewayRequest()
	req.Name = ngw.Name
	req.VpcId = *ngw.VpcId
	req.VSwitchId = ngw.AvailableVSwitches[0]
	req.NatType = "Enhanced"
	resp, err := callApi(c.vpcClient.CreateNatGateway, req)
	if err != nil {
		return nil, err
	}

	var created *NatGateway
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		created, err = c.GetNatGateway(ctx, resp.NatGatewayId)
		if err != nil {
			return false, err
		}
		if created == nil {
			return false, nil
		}
		if *created.Status != "Available" {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return created, nil
}

func (c *actor) GetNatGateway(_ context.Context, id string) (*NatGateway, error) {
	return c.getNatGateway(id)
}

func (c *actor) getNatGateway(id string) (*NatGateway, error) {
	req := vpc.CreateDescribeNatGatewaysRequest()
	req.NatGatewayId = id
	resp, err := c.describeNatGateway(req)

	return single(resp, err)
}

func (c *actor) ListNatGateways(_ context.Context, ids []string) ([]*NatGateway, error) {
	return listByIds(c.getNatGateway, ids)
}

func (c *actor) ListNatGatewaysByVSwitchInVPC(_ context.Context, vpcId string, vswitchIds []string) ([]*NatGateway, error) {
	var ngwList []*NatGateway
	req := vpc.CreateDescribeNatGatewaysRequest()
	req.VpcId = vpcId

	resp, err := c.describeNatGateway(req)
	if err != nil {
		return ngwList, err
	}
	for _, ngw := range resp {
		if contains(vswitchIds, *ngw.VswitchId) {
			ngwList = append(ngwList, ngw)
		}
	}
	return ngwList, nil
}

func (c *actor) FindNatGatewayByVPC(_ context.Context, vpcId string) (*NatGateway, error) {
	req := vpc.CreateDescribeNatGatewaysRequest()
	req.VpcId = vpcId

	resp, err := c.describeNatGateway(req)
	if err != nil {
		return nil, err
	}
	if len(resp) != 1 {
		return nil, fmt.Errorf("count of natgateway is not 1")
	}
	return resp[0], nil
}

func (c *actor) DeleteNatGateway(ctx context.Context, id string) error {
	current, err := c.GetNatGateway(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return nil
	}
	req := vpc.CreateDeleteNatGatewayRequest()
	req.NatGatewayId = id
	req.Force = requests.NewBoolean(true)
	_, err = callApi(c.vpcClient.DeleteNatGateway, req)
	if err != nil {
		return err
	}
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		current, err := c.GetNatGateway(ctx, id)
		if err != nil {
			return false, err
		}
		if current == nil {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return err
	}
	return nil
}

func (c *actor) FindNatGatewayByTags(ctx context.Context, tags Tags) ([]*NatGateway, error) {
	req := vpc.CreateListTagResourcesRequest()
	req.ResourceType = "NATGATEWAY"

	var reqTag []vpc.ListTagResourcesTag
	for k, v := range tags {
		reqTag = append(reqTag, vpc.ListTagResourcesTag{Key: k, Value: v})
	}
	req.Tag = &reqTag

	idList, err := c.listVpcTagResources(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.ListNatGateways(ctx, idList)
}

func (c *actor) DeleteVSwitch(ctx context.Context, id string) error {
	current, err := c.GetVSwitch(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return nil
	}
	req := vpc.CreateDeleteVSwitchRequest()
	req.VSwitchId = id

	_, err = callApi(c.vpcClient.DeleteVSwitch, req)
	if err != nil {
		return err
	}
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		current, err := c.GetVSwitch(ctx, id)
		if err != nil {
			return false, err
		}
		if current == nil {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *actor) CreateVSwitch(ctx context.Context, vsw *VSwitch) (*VSwitch, error) {
	req := vpc.CreateCreateVSwitchRequest()
	req.VSwitchName = vsw.Name
	req.VpcId = *vsw.VpcId
	req.CidrBlock = vsw.CidrBlock
	req.ZoneId = vsw.ZoneId

	resp, err := callApi(c.vpcClient.CreateVSwitch, req)

	if err != nil {
		return nil, err
	}

	var created *VSwitch
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		created, err = c.GetVSwitch(ctx, resp.VSwitchId)
		if err != nil {
			return false, err
		}
		if created == nil {
			return false, nil
		}
		if *created.Status != "Available" {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return created, nil
}

func (c *actor) GetVSwitch(_ context.Context, id string) (*VSwitch, error) {
	return c.getVSwitch(id)
}

func (c *actor) getVSwitch(id string) (*VSwitch, error) {
	req := vpc.CreateDescribeVSwitchesRequest()
	req.VSwitchId = id
	resp, err := c.describeVSwitches(req)
	return single(resp, err)
}

func (c *actor) ListVSwitches(_ context.Context, ids []string) ([]*VSwitch, error) {
	return listByIds(c.getVSwitch, ids)
}

func (c *actor) FindVSwitchesByTags(ctx context.Context, tags Tags) ([]*VSwitch, error) {
	req := vpc.CreateListTagResourcesRequest()
	req.ResourceType = "VSWITCH"

	var reqTag []vpc.ListTagResourcesTag
	for k, v := range tags {
		reqTag = append(reqTag, vpc.ListTagResourcesTag{Key: k, Value: v})
	}
	req.Tag = &reqTag

	idList, err := c.listVpcTagResources(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.ListVSwitches(ctx, idList)
}

func (c *actor) FindVSwitchesByVPC(_ context.Context, vpcId string) ([]*VSwitch, error) {
	req := vpc.CreateDescribeVSwitchesRequest()
	req.VpcId = vpcId

	resp, err := c.describeVSwitches(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *actor) ListVpcs(_ context.Context, ids []string) ([]*VPC, error) {
	return listByIds(c.getVpc, ids)
}

func (c *actor) DeleteVpc(ctx context.Context, id string) error {
	current, err := c.GetVpc(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return nil
	}
	req := vpc.CreateDeleteVpcRequest()
	req.VpcId = id
	req.ForceDelete = requests.NewBoolean(true)

	_, err = callApi(c.vpcClient.DeleteVpc, req)
	if err != nil {
		return err
	}
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		current, err := c.GetVpc(ctx, id)
		if err != nil {
			return false, err
		}
		if current == nil {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *actor) CreateVpc(ctx context.Context, desired *VPC) (*VPC, error) {
	req := vpc.CreateCreateVpcRequest()
	req.VpcName = desired.Name
	req.CidrBlock = desired.CidrBlock

	resp, err := callApi(c.vpcClient.CreateVpc, req)
	if err != nil {
		return nil, err
	}

	var created *VPC
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		created, err = c.GetVpc(ctx, resp.VpcId)
		if err != nil {
			return false, err
		}
		if created == nil {
			return false, nil
		}
		if *created.Status != "Available" {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return created, nil
}

func (c *actor) GetVpc(_ context.Context, id string) (*VPC, error) {
	return c.getVpc(id)
}

func (c *actor) getVpc(id string) (*VPC, error) {
	req := vpc.CreateDescribeVpcsRequest()
	req.VpcId = id

	resp, err := c.describeVpcs(req)
	return single(resp, err)
}

func (c *actor) FindVpcsByTags(ctx context.Context, tags Tags) ([]*VPC, error) {
	req := vpc.CreateListTagResourcesRequest()
	req.ResourceType = "VPC"

	var reqTag []vpc.ListTagResourcesTag
	for k, v := range tags {
		reqTag = append(reqTag, vpc.ListTagResourcesTag{Key: k, Value: v})
	}
	req.Tag = &reqTag

	// var vpcList []*VPC
	idList, err := c.listVpcTagResources(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.ListVpcs(ctx, idList)
}

func (c *actor) createVpcTags(resources []string, tags Tags, resourceType string) error {
	req := vpc.CreateTagResourcesRequest()
	req.ResourceType = resourceType
	req.ResourceId = &resources

	var reqTag []vpc.TagResourcesTag
	for k, v := range tags {
		reqTag = append(reqTag, vpc.TagResourcesTag{Key: k, Value: v})
	}
	req.Tag = &reqTag

	_, err := callApi(c.vpcClient.TagResources, req)
	return err
}

func (c *actor) deleteVpcTags(resources []string, tags Tags, resourceType string) error {
	req := vpc.CreateUnTagResourcesRequest()
	req.ResourceType = resourceType
	req.ResourceId = &resources

	var reqTag []string
	for k := range tags {
		reqTag = append(reqTag, k)
	}
	req.TagKey = &reqTag
	_, err := callApi(c.vpcClient.UnTagResources, req)
	return err
}

func (c *actor) listEcsTagResources(_ context.Context, req *ecs.ListTagResourcesRequest) ([]string, error) {
	var idList []string

	respList, err := page_call(c.ecsClient.ListTagResources, req)
	if err != nil {
		return nil, err
	}

	var theList []ecs.TagResource
	for _, resp := range respList {
		theList = append(theList, resp.TagResources.TagResource...)
	}
	for _, item := range theList {
		if !contains(idList, item.ResourceId) {
			idList = append(idList, item.ResourceId)
		}
	}

	return idList, nil
}

func (c *actor) listVpcTagResources(_ context.Context, req *vpc.ListTagResourcesRequest) ([]string, error) {
	var idList []string

	respList, err := page_call(c.vpcClient.ListTagResources, req)
	if err != nil {
		return nil, err
	}

	var theList []vpc.TagResource
	for _, resp := range respList {
		theList = append(theList, resp.TagResources.TagResource...)
	}
	for _, item := range theList {
		if !contains(idList, item.ResourceId) {
			idList = append(idList, item.ResourceId)
		}
	}

	return idList, nil
}

func (c *actor) describeSecurityGroup(req *ecs.DescribeSecurityGroupsRequest) ([]*SecurityGroup, error) {
	var objList []*SecurityGroup

	respList, err := page_call(c.ecsClient.DescribeSecurityGroups, req)
	if err != nil {
		return nil, err
	}
	var theList []ecs.SecurityGroup
	for _, resp := range respList {
		theList = append(theList, resp.SecurityGroups.SecurityGroup...)
	}

	for _, item := range theList {
		entry, err := c.fromSecurityGroup(item)
		if err == nil && entry != nil {
			objList = append(objList, entry)
		}
	}

	return objList, nil
}

func (c *actor) describeSNATEntry(req *vpc.DescribeSnatTableEntriesRequest) ([]*SNATEntry, error) {
	var entryList []*SNATEntry

	respList, err := page_call(c.vpcClient.DescribeSnatTableEntries, req)
	if err != nil {
		return nil, err
	}
	var theList []vpc.SnatTableEntry
	for _, resp := range respList {
		theList = append(theList, resp.SnatTableEntries.SnatTableEntry...)
	}

	for _, item := range theList {
		entry, err := c.fromSNATEntry(item)
		if err == nil && entry != nil {
			entryList = append(entryList, entry)
		}
	}

	return entryList, nil
}

func (c *actor) describeEIP(req *vpc.DescribeEipAddressesRequest) ([]*EIP, error) {
	var eipList []*EIP

	respList, err := page_call(c.vpcClient.DescribeEipAddresses, req)
	if err != nil {
		return nil, err
	}
	var theList []vpc.EipAddress
	for _, resp := range respList {
		theList = append(theList, resp.EipAddresses.EipAddress...)
	}

	for _, item := range theList {
		eip, err := c.fromEip(item)
		if err == nil && eip != nil {
			eipList = append(eipList, eip)
		}
	}

	return eipList, nil
}

func (c *actor) describeNatGateway(req *vpc.DescribeNatGatewaysRequest) ([]*NatGateway, error) {
	var ngwList []*NatGateway

	respList, err := page_call(c.vpcClient.DescribeNatGateways, req)
	if err != nil {
		return nil, err
	}
	var theList []vpc.NatGateway
	for _, resp := range respList {
		theList = append(theList, resp.NatGateways.NatGateway...)
	}

	for _, item := range theList {
		ngw, err := c.fromNatGateway(item)
		if err == nil && ngw != nil {
			ngwList = append(ngwList, ngw)
		}
	}

	return ngwList, nil
}

func (c *actor) describeVpcs(req *vpc.DescribeVpcsRequest) ([]*VPC, error) {
	var vpcList []*VPC

	respList, err := page_call(c.vpcClient.DescribeVpcs, req)
	if err != nil {
		return nil, err
	}
	var theList []vpc.Vpc
	for _, resp := range respList {
		theList = append(theList, resp.Vpcs.Vpc...)
	}

	for _, item := range theList {
		vpc, err := c.fromVpc(item)
		if err == nil && vpc != nil {
			vpcList = append(vpcList, vpc)
		}
	}

	return vpcList, nil
}

func (c *actor) describeVSwitches(req *vpc.DescribeVSwitchesRequest) ([]*VSwitch, error) {
	var vswitchList []*VSwitch

	respList, err := page_call(c.vpcClient.DescribeVSwitches, req)
	if err != nil {
		return nil, err
	}
	var theList []vpc.VSwitch
	for _, resp := range respList {
		theList = append(theList, resp.VSwitches.VSwitch...)
	}

	for _, item := range theList {
		vswitch, err := c.fromVSwitch(item)
		if err == nil && vswitch != nil {
			vswitchList = append(vswitchList, vswitch)
		}
	}

	return vswitchList, nil
}

func (c *actor) fromSecurityGroupRule(item ecs.Permission) (*SecurityGroupRule, error) {
	rule := &SecurityGroupRule{
		SecurityGroupRuleId: item.SecurityGroupRuleId,
		Policy:              item.Policy,
		Priority:            item.Priority,
		IpProtocol:          item.IpProtocol,
		PortRange:           item.PortRange,
		DestCidrIp:          item.DestCidrIp,
		SourceCidrIp:        item.SourceCidrIp,
		Direction:           item.Direction,
	}
	return rule, nil
}

func (c *actor) fromSecurityGroup(item ecs.SecurityGroup) (*SecurityGroup, error) {
	sg := &SecurityGroup{
		Name:            item.SecurityGroupName,
		VpcId:           item.VpcId,
		SecurityGroupId: item.SecurityGroupId,
	}
	tags := Tags{}
	for _, t := range item.Tags.Tag {
		tags[t.Key] = t.Value
	}
	sg.Tags = tags
	return sg, nil
}

func (c *actor) fromVSwitch(item vpc.VSwitch) (*VSwitch, error) {
	vswitch := &VSwitch{
		Name:      item.VSwitchName,
		VpcId:     &item.VpcId,
		ZoneId:    item.ZoneId,
		CidrBlock: item.CidrBlock,
		Status:    &item.Status,
		VSwitchId: item.VSwitchId,
	}
	tags := Tags{}
	for _, t := range item.Tags.Tag {
		tags[t.Key] = t.Value
	}
	vswitch.Tags = tags
	return vswitch, nil
}

func (c *actor) fromNatGateway(item vpc.NatGateway) (*NatGateway, error) {
	ngw := &NatGateway{
		Name:         item.Name,
		NatGatewayId: item.NatGatewayId,
		VpcId:        &item.VpcId,
		Status:       &item.Status,
		VswitchId:    &item.NatGatewayPrivateInfo.VswitchId,
	}
	tags := Tags{}
	for _, t := range item.Tags.Tag {
		tags[t.Key] = t.Value
	}
	ngw.Tags = tags

	snatTableId := []string{}
	snatTableId = append(snatTableId, item.SnatTableIds.SnatTableId...)

	ngw.SNATTableIDs = snatTableId

	return ngw, nil
}

func (c *actor) fromSNATEntry(item vpc.SnatTableEntry) (*SNATEntry, error) {
	entry := &SNATEntry{
		Name:        item.SnatEntryName,
		VSwitchId:   item.SourceVSwitchId,
		IpAddress:   item.SnatIp,
		SnatTableId: item.SnatTableId,
		SnatEntryId: item.SnatEntryId,
		Status:      &item.Status,
	}
	return entry, nil
}

func (c *actor) fromEip(item vpc.EipAddress) (*EIP, error) {
	eip := &EIP{
		Name:               item.Name,
		Bandwidth:          item.Bandwidth,
		InternetChargeType: item.InternetChargeType,
		EipId:              item.AllocationId,
		Status:             &item.Status,
		InstanceType:       &item.InstanceType,
		InstanceId:         &item.InstanceId,
		IpAddress:          item.IpAddress,
	}
	tags := Tags{}
	for _, t := range item.Tags.Tag {
		tags[t.Key] = t.Value
	}
	eip.Tags = tags
	return eip, nil
}

func (c *actor) fromVpc(item vpc.Vpc) (*VPC, error) {
	vpc := &VPC{
		Name:  item.VpcName,
		VpcId: item.VpcId,

		CidrBlock: item.CidrBlock,
		Status:    &item.Status,
	}

	tags := Tags{}
	for _, t := range item.Tags.Tag {
		tags[t.Key] = t.Value
	}
	vpc.Tags = tags

	return vpc, nil
}

func listByIds[RESP any](geter func(id string) (*RESP, error), ids []string) ([]*RESP, error) {
	var theList []*RESP
	for _, id := range ids {
		obj, err := geter(id)
		if err != nil {
			return nil, err
		}
		if obj != nil {
			theList = append(theList, obj)
		}
	}
	return theList, nil
}

func callApi[REQ any, RESP any](call func(req *REQ) (*RESP, error), req *REQ) (*RESP, error) {
	var resp *RESP
	var err error
	retry_error_code_list := []string{
		"Throttling.User",
		"TaskConflict",
	}
	try_count := 0
	for {
		need_try := false
		try_count++
		cleanQueryParam(req)
		resp, err = call(req)
		if err != nil {
			if serverErr, ok := err.(*alierrors.ServerError); ok {
				if contains(retry_error_code_list, serverErr.ErrorCode()) && try_count < 5 {
					need_try = true
				}
			}
		}
		if !need_try {
			break
		}
		time.Sleep(5 * time.Second)
	}
	return resp, err
}
func page_call[REQ any, RESP any](call func(req *REQ) (*RESP, error), req *REQ) ([]RESP, error) {
	type1_req_type_name_list := []string{
		"DescribeVpcsRequest",
		"DescribeVSwitchesRequest",
		"DescribeNatGatewaysRequest",
		"DescribeEipAddressesRequest",
		"DescribeSnatTableEntriesRequest",
	}
	type2_req_type_name_list := []string{
		"ListTagResourcesRequest",
		"DescribeSecurityGroupsRequest",
	}

	reqTypeName := reflect.ValueOf(req).Elem().Type().Name()
	if contains(type1_req_type_name_list, reqTypeName) {
		return page_call_type_1(call, req)
	} else if contains(type2_req_type_name_list, reqTypeName) {
		return page_call_type_2(call, req)
	}
	return nil, fmt.Errorf("can not found suitable describe function for %s", reqTypeName)
}

func page_call_type_1[REQ any, RESP any](call func(req *REQ) (*RESP, error), req *REQ) ([]RESP, error) {
	var theList []RESP
	const PAGE_SIZE = 10
	var cur_page = 1
	reflect.ValueOf(req).Elem().FieldByName("PageSize").SetString(strconv.Itoa(PAGE_SIZE))
	for {
		reflect.ValueOf(req).Elem().FieldByName("PageNumber").SetString(strconv.Itoa(cur_page))
		resp, err := callApi(call, req)
		if err != nil {
			return nil, err
		}
		theList = append(theList, *resp)

		total := int(reflect.ValueOf(*resp).FieldByName("TotalCount").Int())
		total_page := total / PAGE_SIZE
		remainder := total % PAGE_SIZE
		if remainder > 0 {
			total_page++
		}
		if cur_page >= total_page {
			break
		}
		cur_page++
	}
	return theList, nil
}

func page_call_type_2[REQ any, RESP any](call func(req *REQ) (*RESP, error), req *REQ) ([]RESP, error) {
	var theList []RESP

	resp, err := callApi(call, req)

	if err != nil {
		return nil, err
	}
	theList = append(theList, *resp)

	for {
		nextToken := reflect.ValueOf(*resp).FieldByName("NextToken").String()
		if nextToken == "" {
			break
		}
		reflect.ValueOf(req).Elem().FieldByName("NextToken").SetString(nextToken)
		resp, err := callApi(call, req)
		if err == nil {
			theList = append(theList, *resp)
		}
	}

	return theList, nil
}

func cleanQueryParam(theReq interface{}) {
	if req, ok := theReq.(*requests.RpcRequest); ok {
		queryParam := req.GetQueryParams()
		delete(queryParam, "Version")
		delete(queryParam, "Action")
		delete(queryParam, "Format")
		delete(queryParam, "Timestamp")
		delete(queryParam, "SignatureMethod")
		delete(queryParam, "SignatureType")
		delete(queryParam, "SignatureVersion")
		delete(queryParam, "SignatureNonce")
		delete(queryParam, "AccessKeyId")
		delete(queryParam, "RegionId")
	}
}

func single[T any](list []*T, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

func contains(elems []string, elem string) bool {
	for _, e := range elems {
		if e == elem {
			return true
		}
	}
	return false
}
