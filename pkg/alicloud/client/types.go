// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package client

import (
	"context"
	"sync"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	ram "github.com/aliyun/alibaba-cloud-sdk-go/services/resourcemanager"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/go-logr/logr"
	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ros "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client/ros"
)

// DefaultInternetChargeType is used for EIP
const DefaultInternetChargeType = "PayByTraffic"

// ClientFactory is the new factory to instantiate Alicloud clients.
type ClientFactory interface {
	NewECSClient(region, accessKeyID, accessKeySecret string) (ECS, error)
	NewSTSClient(region, accessKeyID, accessKeySecret string) (STS, error)
	NewSLBClient(region, accessKeyID, accessKeySecret string) (SLB, error)
	NewVPCClient(region, accessKeyID, accessKeySecret string) (VPC, error)
	NewRAMClient(region, accessKeyID, accessKeySecret string) (RAM, error)
	NewROSClient(region, accessKeyID, accessKeySecret string) (ROS, error)
	NewOSSClient(endpoint, accessKeyID, accessKeySecret string) (OSS, error)
	NewOSSClientFromSecretRef(ctx context.Context, client client.Client, secretRef *corev1.SecretReference, region string) (OSS, error)
	NewDNSClient(region, accessKeyID, accessKeySecret string) (DNS, error)
}

// ecsClient implements the ECS interface.
type ecsClient struct {
	ecs.Client
}

// ECS is an interface which declares ECS related methods.
type ECS interface {
	CheckIfImageExists(imageID string) (bool, error)
	CheckIfImageOwnedByAliCloud(imageID string) (bool, error)
	ShareImageToAccount(ctx context.Context, regionID, imageID, accountID string) error
	GetSecurityGroup(name string) (*ecs.DescribeSecurityGroupsResponse, error)
	GetSecurityGroupWithID(id string) (*ecs.DescribeSecurityGroupsResponse, error)
	DescribeSecurityGroups(request *ecs.DescribeSecurityGroupsRequest) (*ecs.DescribeSecurityGroupsResponse, error)
	DescribeSecurityGroupAttribute(request *ecs.DescribeSecurityGroupAttributeRequest) (*ecs.DescribeSecurityGroupAttributeResponse, error)
	DescribeKeyPairs(request *ecs.DescribeKeyPairsRequest) (*ecs.DescribeKeyPairsResponse, error)
	DetachECSInstancesFromSSHKeyPair(keyName string) error
	GetInstances(name string) (*ecs.DescribeInstancesResponse, error)
	GetInstanceType(core int, zoneID string) (*ecs.DescribeAvailableResourceResponse, error)
	CreateInstances(instanceName, securityGroupID, imageID, vSwitchId, zoneID, instanceTypeID, userData string) (*ecs.RunInstancesResponse, error)
	DeleteInstances(id string, force bool) error
	CreateSecurityGroups(vpcId, name string) (*ecs.CreateSecurityGroupResponse, error)
	DeleteSecurityGroups(id string) error
	AllocatePublicIp(id string) (*ecs.AllocatePublicIpAddressResponse, error)
	CreateIngressRule(request *ecs.AuthorizeSecurityGroupRequest) error
	CreateEgressRule(request *ecs.AuthorizeSecurityGroupEgressRequest) error
	RevokeIngressRule(request *ecs.RevokeSecurityGroupRequest) error
	RevokeEgressRule(request *ecs.RevokeSecurityGroupEgressRequest) error
}

// stsClient implements the STS interface.
type stsClient struct {
	sts.Client
}

// STS is an interface which declares STS related methods.
type STS interface {
	GetAccountIDFromCallerIdentity(ctx context.Context) (string, error)
}

// slbClient implements the SLB interface.
type slbClient struct {
	slb.Client
}

// SLB is an interface which declares SLB related methods.
type SLB interface {
	GetLoadBalancerIDs(ctx context.Context, region string) ([]string, error)
	GetFirstVServerGroupName(ctx context.Context, region, loadBalancerID string) (string, error)
	DeleteLoadBalancer(ctx context.Context, region, loadBalancerID string) error
	SetLoadBalancerDeleteProtection(ctx context.Context, region, loadBalancerID string, protection bool) error
}

// vpcClient implements the VPC interface.
type vpcClient struct {
	vpc.Client
}

// VPC is an interface which declares VPC related methods.
type VPC interface {
	GetVPCWithID(ctx context.Context, vpcID string) ([]vpc.Vpc, error)
	GetNatGatewaysWithVPCID(ctx context.Context, vpcID string) ([]vpc.NatGateway, error)
	GetEIPWithID(ctx context.Context, eipID string) ([]vpc.EipAddress, error)
	GetEnhanhcedNatGatewayAvailableZones(ctx context.Context, region string) ([]string, error)
	GetVPCInfo(ctx context.Context, vpcID string) (*VPCInfo, error)
	GetVPCInfoByName(name string) (*VPCInfo, error)
	FetchEIPInternetChargeType(ctx context.Context, natGateway *vpc.NatGateway, vpcID string) (string, error)

	CreateVpc(request *vpc.CreateVpcRequest) (response *vpc.CreateVpcResponse, err error)
	DescribeVpcs(request *vpc.DescribeVpcsRequest) (response *vpc.DescribeVpcsResponse, err error)
	TagResources(request *vpc.TagResourcesRequest) (response *vpc.TagResourcesResponse, err error)
	UnTagResources(request *vpc.UnTagResourcesRequest) (response *vpc.UnTagResourcesResponse, err error)
	ListTagResources(request *vpc.ListTagResourcesRequest) (response *vpc.ListTagResourcesResponse, err error)
	DeleteVpc(request *vpc.DeleteVpcRequest) (response *vpc.DeleteVpcResponse, err error)
	CreateVSwitch(request *vpc.CreateVSwitchRequest) (response *vpc.CreateVSwitchResponse, err error)
	DescribeVSwitches(request *vpc.DescribeVSwitchesRequest) (response *vpc.DescribeVSwitchesResponse, err error)
	GetVSwitchesInfoByID(name string) (*VSwitchInfo, error)
	DeleteVSwitch(request *vpc.DeleteVSwitchRequest) (response *vpc.DeleteVSwitchResponse, err error)
	CreateNatGateway(request *vpc.CreateNatGatewayRequest) (response *vpc.CreateNatGatewayResponse, err error)
	DescribeNatGateways(request *vpc.DescribeNatGatewaysRequest) (response *vpc.DescribeNatGatewaysResponse, err error)
	DeleteNatGateway(request *vpc.DeleteNatGatewayRequest) (response *vpc.DeleteNatGatewayResponse, err error)
	DescribeSnatTableEntries(request *vpc.DescribeSnatTableEntriesRequest) (response *vpc.DescribeSnatTableEntriesResponse, err error)
	DescribeEipAddresses(request *vpc.DescribeEipAddressesRequest) (response *vpc.DescribeEipAddressesResponse, err error)

	AllocateEipAddress(request *vpc.AllocateEipAddressRequest) (response *vpc.AllocateEipAddressResponse, err error)
	ReleaseEipAddress(request *vpc.ReleaseEipAddressRequest) (response *vpc.ReleaseEipAddressResponse, err error)
	ModifyEipAddressAttribute(request *vpc.ModifyEipAddressAttributeRequest) (response *vpc.ModifyEipAddressAttributeResponse, err error)
	AssociateEipAddress(request *vpc.AssociateEipAddressRequest) (response *vpc.AssociateEipAddressResponse, err error)
	UnassociateEipAddress(request *vpc.UnassociateEipAddressRequest) (response *vpc.UnassociateEipAddressResponse, err error)
	CreateSnatEntry(request *vpc.CreateSnatEntryRequest) (response *vpc.CreateSnatEntryResponse, err error)
	DeleteSnatEntry(request *vpc.DeleteSnatEntryRequest) (response *vpc.DeleteSnatEntryResponse, err error)
}

// ramClient implements the RAM interface.
type ramClient struct {
	ram.Client
}

// RAM is an interface which declares RAM related methods.
type RAM interface {
	CreateServiceLinkedRole(regionID, serviceName string) error
	GetServiceLinkedRole(roleName string) (*ram.Role, error)
}

// ROS is an interface which declares ROS related methods.
type ROS interface {
	ListStacks(request *ros.ListStacksRequest) (response *ros.ListStacksResponse, err error)
	GetStack(request *ros.GetStackRequest) (response *ros.GetStackResponse, err error)
	CreateStack(request *ros.CreateStackRequest) (response *ros.CreateStackResponse, err error)
	DeleteStack(request *ros.DeleteStackRequest) (response *ros.DeleteStackResponse, err error)
}

// ossClient implements the OSS interface.
type ossClient struct {
	oss.Client
}

// OSS is an interface which declares OSS related methods.
type OSS interface {
	DeleteObjectsWithPrefix(ctx context.Context, bucketName, prefix string) error
	CreateBucketIfNotExists(ctx context.Context, bucketName string) error
	DeleteBucketIfExists(ctx context.Context, bucketName string) error
}

// VPCInfo contains info about an existing VPC.
type VPCInfo struct {
	CIDR               string
	NATGatewayID       string
	SNATTableIDs       string
	InternetChargeType string
	VSwitchID          string
	VPCID              string
}

// VSwitchInfo contains info about an existing VSwitchInfo.
type VSwitchInfo struct {
	ZoneID string
}

// dnsClient implements the DNS interface.
type dnsClient struct {
	alidns.Client
	accessKeyID            string
	domainsCache           *cache.Expiring
	domainsCacheMutex      *sync.Mutex
	RateLimiter            *rate.Limiter
	RateLimiterWaitTimeout time.Duration
	Logger                 logr.Logger
}

// DNS is an interface which declares DNS related methods.
type DNS interface {
	GetDomainNames(context.Context) (map[string]string, error)
	GetDomainName(context.Context, string) (string, error)
	CreateOrUpdateDomainRecords(context.Context, string, string, string, []string, int64) error
	DeleteDomainRecords(context.Context, string, string, string) error
}
