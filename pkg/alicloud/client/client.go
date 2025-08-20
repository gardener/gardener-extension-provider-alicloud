// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	ram "github.com/aliyun/alibaba-cloud-sdk-go/services/resourcemanager"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client/ros"
)

// ComputeStorageEndpoint computes the OSS storage endpoint based on the given region.
func ComputeStorageEndpoint(region string) string {
	return fmt.Sprintf("https://oss-%s.aliyuncs.com/", region)
}

type clientFactory struct {
	limit             rate.Limit
	burst             int
	waitTimeout       time.Duration
	rateLimiters      *cache.Expiring
	rateLimitersMutex sync.Mutex
	domainsCache      *cache.Expiring
	domainsCacheMutex sync.Mutex
}

// NewClientFactory creates a new clientFactory instance that can be used to instantiate Alicloud clients.
func NewClientFactory() ClientFactory {
	return &clientFactory{
		domainsCache: cache.NewExpiring(),
	}
}

// NewClientFactoryWithRateLimit creates a new clientFactory instance that can be used to instantiate Alicloud dns clients.
func NewClientFactoryWithRateLimit(limit rate.Limit, burst int, waitTimeout time.Duration) ClientFactory {
	return &clientFactory{
		limit:        limit,
		burst:        burst,
		waitTimeout:  waitTimeout,
		rateLimiters: cache.NewExpiring(),
		domainsCache: cache.NewExpiring(),
	}
}

// NewOSSClient creates an new OSS client with given endpoint, accessKeyID, and accessKeySecret.
func (f *clientFactory) NewOSSClient(endpoint, accessKeyID, accessKeySecret string) (OSS, error) {
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}

	return &ossClient{
		*client,
	}, nil
}

// NewOSSClientFromSecretRef creates a new OSS Client using the credentials from <secretRef>.
func (f *clientFactory) NewOSSClientFromSecretRef(ctx context.Context, client client.Client, secretRef *corev1.SecretReference, region string) (OSS, error) {
	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, client, secretRef)
	if err != nil {
		return nil, err
	}

	return f.NewOSSClient(ComputeStorageEndpoint(region), credentials.AccessKeyID, credentials.AccessKeySecret)
}

// DeleteObjectsWithPrefix deletes the OSS objects with the specific <prefix> from <bucketName>.
// If it does not exist, no error is returned.
func (c *ossClient) DeleteObjectsWithPrefix(ctx context.Context, bucketName, prefix string) error {
	bucket, err := c.Bucket(bucketName)
	if err != nil {
		return err
	}

	var expirationOption oss.Option
	t, ok := ctx.Deadline()
	if ok {
		expirationOption = oss.Expires(t)
	}

	marker := ""
	for {
		lsRes, err := bucket.ListObjects(oss.Marker(marker), oss.Prefix(prefix), oss.MaxKeys(1000), expirationOption)

		if err != nil {
			return err
		}

		var objectKeys []string
		for _, object := range lsRes.Objects {
			objectKeys = append(objectKeys, object.Key)
		}

		if len(objectKeys) != 0 {
			if _, err := bucket.DeleteObjects(objectKeys, oss.DeleteObjectsQuiet(true), expirationOption); err != nil {
				return err
			}
		}

		if !lsRes.IsTruncated {
			return nil
		}
		marker = lsRes.NextMarker
	}
}

// CreateBucketIfNotExists creates the OSS bucket with name <bucketName> in <region>. If it already exist,
// no error is returned.
func (c *ossClient) CreateBucketIfNotExists(ctx context.Context, bucketName string) error {
	var expirationOption oss.Option
	t, ok := ctx.Deadline()
	if ok {
		expirationOption = oss.Expires(t)
	}

	if err := c.CreateBucket(bucketName, oss.StorageClass(oss.StorageStandard), expirationOption); err != nil {
		if ossErr, ok := err.(oss.ServiceError); !ok {
			return err
		} else if ossErr.StatusCode != http.StatusConflict {
			return err
		}
	}

	encryptionRule := oss.ServerEncryptionRule{
		SSEDefault: oss.SSEDefaultRule{
			SSEAlgorithm: string(oss.AESAlgorithm),
		},
	}
	if err := c.SetBucketEncryption(bucketName, encryptionRule, expirationOption); err != nil {
		return err
	}

	rules := []oss.LifecycleRule{
		{
			Prefix: "",
			Status: "Enabled",
			AbortMultipartUpload: &oss.LifecycleAbortMultipartUpload{
				Days: 7,
			},
		},
	}
	return c.SetBucketLifecycle(bucketName, rules)
}

// GetBucketInfo retrieves bucket details.
func (c *ossClient) GetBucketInfo(_ context.Context, bucketName string) (*oss.BucketInfo, error) {
	result, err := c.Client.GetBucketInfo(bucketName)
	if err != nil {
		return nil, err
	}

	return &result.BucketInfo, nil
}

// DeleteBucketIfExists deletes the Alicloud OSS bucket with name <bucketName>. If it does not exist,
// no error is returned.
func (c *ossClient) DeleteBucketIfExists(ctx context.Context, bucketName string) error {
	if err := c.DeleteBucket(bucketName); err != nil {
		if ossErr, ok := err.(oss.ServiceError); ok {
			switch ossErr.StatusCode {
			case http.StatusNotFound:
				return nil

			case http.StatusConflict:
				if err := c.DeleteObjectsWithPrefix(ctx, bucketName, ""); err != nil {
					return err
				}
				return c.DeleteBucketIfExists(ctx, bucketName)

			default:
				return ossErr
			}
		}
	}
	return nil
}

// NewECSClient creates a new ECS client with given region, accessKeyID, and accessKeySecret.
func (f *clientFactory) NewECSClient(region, accessKeyID, accessKeySecret string) (ECS, error) {
	client, err := ecs.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}

	return &ecsClient{
		*client,
	}, nil
}

// CheckIfImageExists checks whether given imageID can be accessed by the client.
func (c *ecsClient) CheckIfImageExists(imageID string) (bool, error) {
	request := ecs.CreateDescribeImagesRequest()
	request.ImageId = imageID
	request.SetScheme("HTTPS")
	response, err := c.DescribeImages(request)
	if err != nil {
		return false, err
	}
	return response.TotalCount > 0, nil
}

// GetImageInfo returns image metadata by imageID
func (c *ecsClient) GetImageInfo(imageID string) (*ecs.DescribeImagesResponse, error) {
	request := ecs.CreateDescribeImagesRequest()
	request.ImageId = imageID
	request.SetScheme("HTTPS")
	response, err := c.DescribeImages(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetSecurityGroup return security group metadata by security group name
func (c *ecsClient) GetSecurityGroup(name string) (*ecs.DescribeSecurityGroupsResponse, error) {
	request := ecs.CreateDescribeSecurityGroupsRequest()
	request.SetScheme("HTTPS")
	request.SecurityGroupName = name
	return c.DescribeSecurityGroups(request)
}

// GetSecurityGroup return security group metadata by security group name
func (c *ecsClient) GetSecurityGroupWithID(id string) (*ecs.DescribeSecurityGroupsResponse, error) {
	request := ecs.CreateDescribeSecurityGroupsRequest()
	request.SetScheme("HTTPS")
	request.SecurityGroupId = id
	return c.DescribeSecurityGroups(request)
}

// GetInstances return instance metadata by instance name
func (c *ecsClient) GetInstances(name string) (*ecs.DescribeInstancesResponse, error) {
	request := ecs.CreateDescribeInstancesRequest()
	request.SetScheme("HTTPS")
	request.InstanceName = name
	return c.DescribeInstances(request)
}

// GetAvailableInstanceType return metadata of instance type
func (c *ecsClient) GetAvailableInstanceType(cores int, zoneID string) (*ecs.DescribeAvailableResourceResponse, error) {
	request := ecs.CreateDescribeAvailableResourceRequest()
	request.SetScheme("HTTPS")
	request.DestinationResource = "InstanceType"
	request.InstanceChargeType = "PostPaid"
	request.NetworkCategory = "vpc"
	request.Cores = requests.NewInteger(cores)
	request.ZoneId = zoneID
	return c.DescribeAvailableResource(request)
}

// ListAllInstanceType return metadata of instance type
func (c *ecsClient) ListAllInstanceType() (*ecs.DescribeInstanceTypesResponse, error) {
	request := ecs.CreateDescribeInstanceTypesRequest()
	request.SetScheme("HTTPS")
	return c.DescribeInstanceTypes(request)
}

// CreateInstance create a instance
func (c *ecsClient) CreateInstances(instanceName, securityGroupID, imageID, vSwitchId, zoneID, instanceTypeID, userData string) (*ecs.RunInstancesResponse, error) {
	request := ecs.CreateRunInstancesRequest()
	request.SetScheme("HTTPS")
	request.ImageId = imageID
	request.InstanceName = instanceName
	request.SecurityGroupId = securityGroupID
	request.InstanceType = instanceTypeID
	request.ZoneId = zoneID
	request.VSwitchId = vSwitchId
	// assign public IP addresses to the new instances if InternetMaxBandwidthOut parameter to a value greater than 0
	request.InternetMaxBandwidthOut = requests.NewInteger(5)
	request.UserData = userData
	return c.RunInstances(request)
}

// DeleteInstance delete a instance
func (c *ecsClient) DeleteInstances(id string, force bool) error {
	request := ecs.CreateDeleteInstanceRequest()
	request.SetScheme("HTTPS")
	request.InstanceId = id
	request.Force = requests.NewBoolean(force)
	_, err := c.DeleteInstance(request)
	return err
}

// CreateSecurityGroups create a security group
func (c *ecsClient) CreateSecurityGroups(vpcId, name string) (*ecs.CreateSecurityGroupResponse, error) {
	request := ecs.CreateCreateSecurityGroupRequest()
	request.SetScheme("HTTPS")
	request.VpcId = vpcId
	request.SecurityGroupName = name
	return c.CreateSecurityGroup(request)
}

// DeleteSecurityGroups delete a security Group
func (c *ecsClient) DeleteSecurityGroups(id string) error {
	request := ecs.CreateDeleteSecurityGroupRequest()
	request.SetScheme("HTTPS")
	request.SecurityGroupId = id
	_, err := c.DeleteSecurityGroup(request)
	return err
}

// AllocatePublicIp allocate public ip
func (c *ecsClient) AllocatePublicIp(id string) (*ecs.AllocatePublicIpAddressResponse, error) {
	request := ecs.CreateAllocatePublicIpAddressRequest()
	request.SetScheme("HTTPS")
	request.InstanceId = id
	return c.AllocatePublicIpAddress(request)
}

// CreateIngressRule create ingress rule
func (c *ecsClient) CreateIngressRule(request *ecs.AuthorizeSecurityGroupRequest) error {
	_, err := c.AuthorizeSecurityGroup(request)
	return err
}

// CreateEgressRule create egress rule
func (c *ecsClient) CreateEgressRule(request *ecs.AuthorizeSecurityGroupEgressRequest) error {
	_, err := c.AuthorizeSecurityGroupEgress(request)
	return err
}

// RevokeIngressRule revoke ingress rule
func (c *ecsClient) RevokeIngressRule(request *ecs.RevokeSecurityGroupRequest) error {
	_, err := c.RevokeSecurityGroup(request)
	return err
}

// RevokeEgressRule revoke egress rule
func (c *ecsClient) RevokeEgressRule(request *ecs.RevokeSecurityGroupEgressRequest) error {
	_, err := c.RevokeSecurityGroupEgress(request)
	return err
}

// CheckIfImageOwnedByAliCloud checks if the given image ID is owned by AliCloud
func (c *ecsClient) CheckIfImageOwnedByAliCloud(imageID string) (bool, error) {
	request := ecs.CreateDescribeImagesRequest()
	request.ImageId = imageID
	request.SetScheme("HTTPS")
	response, err := c.DescribeImages(request)
	if err != nil {
		return false, err
	}

	if response.TotalCount == 0 {
		return false, fmt.Errorf("image %v is not found", imageID)
	}

	return response.Images.Image[0].ImageOwnerAlias == "system", nil
}

// ShareImageToAccount shares the given image to target account from current client.
func (c *ecsClient) ShareImageToAccount(_ context.Context, regionID, imageID, accountID string) error {
	request := ecs.CreateModifyImageSharePermissionRequest()
	request.RegionId = regionID
	request.ImageId = imageID
	request.AddAccount = &[]string{accountID}
	request.SetScheme("HTTPS")
	_, err := c.ModifyImageSharePermission(request)
	return err
}

// DetachECSInstancesFromSSHKeyPair finds all ECS instances and detach them from the specified SSH key pair.
func (c *ecsClient) DetachECSInstancesFromSSHKeyPair(keyName string) error {
	const pageSize = 50
	describeECSRequest := ecs.CreateDescribeInstancesRequest()
	describeECSRequest.KeyPairName = keyName
	describeECSRequest.PageSize = requests.NewInteger(pageSize)

	var datachInstanceBatches []string
	nextPage := 0
	for {
		nextPage++
		describeECSRequest.PageNumber = requests.NewInteger(nextPage)
		instancesResponse, err := c.DescribeInstances(describeECSRequest)
		if err != nil {
			return err
		}

		if len(instancesResponse.Instances.Instance) == 0 {
			break
		}

		var instanceIDs []string
		for _, instance := range instancesResponse.Instances.Instance {
			instanceIDs = append(instanceIDs, instance.InstanceId)
		}

		jsonRaw, err := json.Marshal(instanceIDs)
		if err != nil {
			return err
		}

		datachInstanceBatches = append(datachInstanceBatches, string(jsonRaw))

		if len(instancesResponse.Instances.Instance) < pageSize {
			break
		}
	}

	detachKeyPairRequest := ecs.CreateDetachKeyPairRequest()
	detachKeyPairRequest.KeyPairName = keyName
	for _, ids := range datachInstanceBatches {
		detachKeyPairRequest.InstanceIds = ids
		detachResponse, err := c.DetachKeyPair(detachKeyPairRequest)
		if err != nil {
			return err
		}
		if detachResponse.FailCount != "0" {
			return fmt.Errorf("failed to detach keypair %s from instances: %v", keyName, detachResponse.Results)
		}
	}

	return nil
}

// NewSTSClient creates a new STS client with given region, accessKeyID, and accessKeySecret.
func (f *clientFactory) NewSTSClient(region, accessKeyID, accessKeySecret string) (STS, error) {
	client, err := sts.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}

	return &stsClient{
		*client,
	}, nil
}

// GetAccountIDFromCallerIdentity gets caller's accountID.
func (c *stsClient) GetAccountIDFromCallerIdentity(_ context.Context) (string, error) {
	request := sts.CreateGetCallerIdentityRequest()
	request.SetScheme("HTTPS")
	response, err := c.GetCallerIdentity(request)
	if err != nil {
		return "", err
	}
	return response.AccountId, nil
}

// NewSLBClient creates a new SLB client with given region, accessKeyID, and accessKeySecret.
func (f *clientFactory) NewSLBClient(region, accessKeyID, accessKeySecret string) (SLB, error) {
	client, err := slb.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}

	return &slbClient{
		*client,
	}, nil
}

// GetLoadBalancerIDs gets LoadBalancerIDs from all LoadBalancers in the given region.
func (c *slbClient) GetLoadBalancerIDs(_ context.Context, region string) ([]string, error) {
	var (
		loadBalancerIDs []string
		pageNumber      = 1
		pageSize        = 100
		request         = slb.CreateDescribeLoadBalancersRequest()
	)
	request.SetScheme("HTTPS")
	request.RegionId = region
	request.PageSize = requests.NewInteger(pageSize)

	for {
		request.PageNumber = requests.NewInteger(pageNumber)
		response, err := c.DescribeLoadBalancers(request)
		if err != nil {
			return nil, err
		}
		for _, loadBalancer := range response.LoadBalancers.LoadBalancer {
			loadBalancerIDs = append(loadBalancerIDs, loadBalancer.LoadBalancerId)
		}

		if pageNumber*pageSize >= response.TotalCount {
			break
		}
		pageNumber++
	}
	return loadBalancerIDs, nil
}

// GetFirstVServerGroupName gets the VServerGroupName of the first VServerGroup in the LoadBalancer with given region and loadBalancerID.
func (c *slbClient) GetFirstVServerGroupName(_ context.Context, region, loadBalancerID string) (string, error) {
	request := slb.CreateDescribeVServerGroupsRequest()
	request.SetScheme("HTTPS")
	request.RegionId = region
	request.LoadBalancerId = loadBalancerID
	response, err := c.DescribeVServerGroups(request)
	if err != nil {
		return "", err
	}
	if len(response.VServerGroups.VServerGroup) == 0 {
		return "", nil
	}
	return response.VServerGroups.VServerGroup[0].VServerGroupName, nil
}

// DeleteLoadBalancer deletes the LoadBalancer with given region and loadBalancerID.
func (c *slbClient) DeleteLoadBalancer(_ context.Context, region, loadBalancerID string) error {
	request := slb.CreateDeleteLoadBalancerRequest()
	request.SetScheme("HTTPS")
	request.RegionId = region
	request.LoadBalancerId = loadBalancerID
	_, err := c.Client.DeleteLoadBalancer(request)
	return err
}

// SetLoadBalancerDeleteProtection sets the protection flag of load balancer with given loadBalancerID.
func (c *slbClient) SetLoadBalancerDeleteProtection(_ context.Context, region, loadBalancerID string, protection bool) error {
	request := slb.CreateSetLoadBalancerDeleteProtectionRequest()

	if protection {
		request.DeleteProtection = "on"
	} else {
		request.DeleteProtection = "off"
	}
	request.SetScheme("HTTPS")
	request.RegionId = region
	request.LoadBalancerId = loadBalancerID
	_, err := c.Client.SetLoadBalancerDeleteProtection(request)

	return err
}

// NewSLBClient creates a new SLB client with given region, accessKeyID, and accessKeySecret.
func (f *clientFactory) NewVPCClient(region, accessKeyID, accessKeySecret string) (VPC, error) {
	client, err := vpc.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}

	return &vpcClient{
		*client,
	}, nil
}

// GetEnhanhcedNatGatewayAvailableZones returns zones in which Enhanced NatGateway is available with given region
func (c *vpcClient) GetEnhanhcedNatGatewayAvailableZones(_ context.Context, region string) ([]string, error) {
	request := vpc.CreateListEnhanhcedNatGatewayAvailableZonesRequest()
	request.RegionId = region
	response, err := c.ListEnhanhcedNatGatewayAvailableZones(request)
	if err != nil {
		return nil, err
	}
	zoneIDs := make([]string, 0, len(response.Zones))
	for _, zone := range response.Zones {
		zoneIDs = append(zoneIDs, zone.ZoneId)
	}
	return zoneIDs, nil
}

// GetVPCWithID returns VPC with given vpcID.
func (c *vpcClient) GetVPCWithID(_ context.Context, vpcID string) ([]vpc.Vpc, error) {
	request := vpc.CreateDescribeVpcsRequest()
	request.VpcId = vpcID
	response, err := c.DescribeVpcs(request)
	if err != nil {
		return nil, err
	}

	return response.Vpcs.Vpc, nil
}

// GetNatGatewaysWithVPCID returns Gateways with given vpcID.
func (c *vpcClient) GetNatGatewaysWithVPCID(_ context.Context, vpcID string) ([]vpc.NatGateway, error) {
	request := vpc.CreateDescribeNatGatewaysRequest()
	request.VpcId = vpcID
	response, err := c.DescribeNatGateways(request)
	if err != nil {
		return nil, err
	}

	return response.NatGateways.NatGateway, nil
}

// GetEIPWithID returns EIP with given eipID
func (c *vpcClient) GetEIPWithID(_ context.Context, eipID string) ([]vpc.EipAddress, error) {
	request := vpc.CreateDescribeEipAddressesRequest()
	request.AllocationId = eipID

	response, err := c.DescribeEipAddresses(request)
	if err != nil {
		return nil, err
	}

	return response.EipAddresses.EipAddress, nil
}

// GetVPCInfo gets info of an existing VPC.
func (c *vpcClient) GetVPCInfo(ctx context.Context, vpcID string) (*VPCInfo, error) {
	vpc, err := c.GetVPCWithID(ctx, vpcID)
	if err != nil {
		return nil, err
	}

	if len(vpc) != 1 {
		return nil, fmt.Errorf("ambiguous VPC response: expected 1 VPC but got %v", vpc)
	}

	natGateways, err := c.GetNatGatewaysWithVPCID(ctx, vpcID)
	if err != nil {
		return nil, err
	}

	if len(natGateways) != 1 {
		return nil, fmt.Errorf("ambiguous NAT Gateway response: expected 1 NAT Gateway but got %v", natGateways)
	}

	natGateway := natGateways[0]
	internetChargeType, err := c.FetchEIPInternetChargeType(ctx, &natGateway, vpcID)
	if err != nil {
		return nil, err
	}

	return &VPCInfo{
		CIDR:               vpc[0].CidrBlock,
		NATGatewayID:       natGateways[0].NatGatewayId,
		SNATTableIDs:       strings.Join(natGateway.SnatTableIds.SnatTableId, ","),
		InternetChargeType: internetChargeType,
	}, nil
}

// GetVPCInfoByName gets info of an existing VPC by VPC name
func (c *vpcClient) GetVPCInfoByName(name string) (*VPCInfo, error) {
	request := vpc.CreateDescribeVpcsRequest()
	request.VpcName = name

	vpc, err := c.DescribeVpcs(request)
	if err != nil {
		return nil, err
	}

	if len(vpc.Vpcs.Vpc) == 0 {
		return nil, fmt.Errorf("shoot vpc must be not empty")
	}

	if len(vpc.Vpcs.Vpc[0].VSwitchIds.VSwitchId) == 0 {
		return nil, fmt.Errorf("vswitch must be not empty")
	}

	return &VPCInfo{
		VSwitchID: vpc.Vpcs.Vpc[0].VSwitchIds.VSwitchId[0],
		VPCID:     vpc.Vpcs.Vpc[0].VpcId,
	}, nil
}

// GetVSwitchesInfoByID get info of VSwitch by ID
func (c *vpcClient) GetVSwitchesInfoByID(id string) (*VSwitchInfo, error) {
	request := vpc.CreateDescribeVSwitchesRequest()
	request.VSwitchId = id
	vswitches, err := c.DescribeVSwitches(request)
	if err != nil {
		return nil, err
	}

	if len(vswitches.VSwitches.VSwitch) == 0 {
		return nil, fmt.Errorf("vswitches not found")
	}

	return &VSwitchInfo{
		ZoneID: vswitches.VSwitches.VSwitch[0].ZoneId,
	}, nil
}

// FetchEIPInternetChargeType fetches the internet charge type for the VPC's EIP.
func (c *vpcClient) FetchEIPInternetChargeType(ctx context.Context, natGateway *vpc.NatGateway, vpcID string) (string, error) {
	if natGateway == nil {
		natGateways, err := c.GetNatGatewaysWithVPCID(ctx, vpcID)
		if err != nil {
			return "", err
		}
		if len(natGateways) != 1 {
			return DefaultInternetChargeType, nil
		}
		natGateway = &natGateways[0]
	}

	if len(natGateway.IpLists.IpList) == 0 {
		return DefaultInternetChargeType, nil
	}

	ipList := natGateway.IpLists.IpList[0]
	eip, err := c.GetEIPWithID(ctx, ipList.AllocationId)
	if err != nil {
		return "", err
	}
	if len(eip) == 0 {
		return DefaultInternetChargeType, nil
	}

	return eip[0].InternetChargeType, nil
}

// NewRAMClient creates a new RAM client with given region, accessKeyID, and accessKeySecret.
func (f *clientFactory) NewRAMClient(region, accessKeyID, accessKeySecret string) (RAM, error) {
	client, err := ram.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}

	return &ramClient{
		*client,
	}, nil
}

// NewROSClient creates a new ROS client with given region, accessKeyID, and accessKeySecret.
func (f *clientFactory) NewROSClient(region, accessKeyID, accessKeySecret string) (ROS, error) {
	return ros.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
}

// GetServiceLinkedRole returns service linked role from Alicloud SDK calls with given role name.
func (c *ramClient) GetServiceLinkedRole(roleName string) (*ram.Role, error) {
	request := ram.CreateGetRoleRequest()
	request.RoleName = roleName
	request.SetScheme("HTTPS")

	response, err := c.GetRole(request)
	if err != nil {
		if isNoPermissionError(err) {
			return nil, fmt.Errorf("no permission to get service linked role, please grant credentials correct privileges. See https://github.com/gardener/gardener-extension-provider-alicloud/blob/v1.21.1/docs/usage-as-end-user.md#Permissions")
		}
		if isRoleNotExistsError(err) {
			return nil, nil
		}
		return nil, err
	}

	if !response.Role.IsServiceLinkedRole {
		return nil, fmt.Errorf("%v exists, but is not a service linked role", roleName)
	}

	return &response.Role, nil
}

// CreateServiceLinkedRole creates service linked role Alicloud SDK calls.
func (c *ramClient) CreateServiceLinkedRole(regionID, serviceName string) error {
	request := ram.CreateCreateServiceLinkedRoleRequest()
	request.ServiceName = serviceName
	request.SetScheme("HTTPS")
	request.RegionId = regionID

	if _, err := c.Client.CreateServiceLinkedRole(request); err != nil {
		if isNoPermissionError(err) {
			return fmt.Errorf("no permission to create service linked role, please grant credentials correct privileges. See https://github.com/gardener/gardener-extension-provider-alicloud/blob/v1.21.1/docs/usage-as-end-user.md#Permissions")
		}
		return err
	}

	return nil
}

func isNoPermissionError(err error) bool {
	if serverError, ok := err.(*errors.ServerError); ok {
		if serverError.ErrorCode() == alicloud.ErrorCodeNoPermission {
			return true
		}
	}
	return false
}

func isRoleNotExistsError(err error) bool {
	if serverError, ok := err.(*errors.ServerError); ok {
		if serverError.ErrorCode() == alicloud.ErrorCodeRoleEntityNotExist {
			return true
		}
	}
	return false
}
