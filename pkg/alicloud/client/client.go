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
	"fmt"
	"net/http"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	ram "github.com/aliyun/alibaba-cloud-sdk-go/services/resourcemanager"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client/ros"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ComputeStorageEndpoint computes the OSS storage endpoint based on the given region.
func ComputeStorageEndpoint(region string) string {
	return fmt.Sprintf("https://oss-%s.aliyuncs.com/", region)
}

type clientFactory struct{}

// NewClientFactory creates a new clientFactory instance that can be used to instantiate Alicloud clients.
func NewClientFactory() ClientFactory {
	return &clientFactory{}
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
	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, client, secretRef, false)
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

		if lsRes.IsTruncated {
			marker = lsRes.NextMarker
		} else {
			return nil
		}
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
func (c *ecsClient) CheckIfImageExists(ctx context.Context, imageID string) (bool, error) {
	request := ecs.CreateDescribeImagesRequest()
	request.ImageId = imageID
	request.SetScheme("HTTPS")
	response, err := c.DescribeImages(request)
	if err != nil {
		return false, err
	}
	return response.TotalCount > 0, nil
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
func (c *ecsClient) ShareImageToAccount(ctx context.Context, regionID, imageID, accountID string) error {
	request := ecs.CreateModifyImageSharePermissionRequest()
	request.RegionId = regionID
	request.ImageId = imageID
	request.AddAccount = &[]string{accountID}
	request.SetScheme("HTTPS")
	_, err := c.ModifyImageSharePermission(request)
	return err
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
func (c *stsClient) GetAccountIDFromCallerIdentity(ctx context.Context) (string, error) {
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
func (c *slbClient) GetLoadBalancerIDs(ctx context.Context, region string) ([]string, error) {
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
func (c *slbClient) GetFirstVServerGroupName(ctx context.Context, region, loadBalancerID string) (string, error) {
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
func (c *slbClient) DeleteLoadBalancer(ctx context.Context, region, loadBalancerID string) error {
	request := slb.CreateDeleteLoadBalancerRequest()
	request.SetScheme("HTTPS")
	request.RegionId = region
	request.LoadBalancerId = loadBalancerID
	_, err := c.Client.DeleteLoadBalancer(request)
	return err
}

// SetLoadBalancerDeleteProtection sets the protection flag of load balancer with given loadBalancerID.
func (c *slbClient) SetLoadBalancerDeleteProtection(ctx context.Context, region, loadBalancerID string, protection bool) error {
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

// GetVPCWithID returns VPC with given vpcID.
func (c *vpcClient) GetVPCWithID(ctx context.Context, vpcID string) ([]vpc.Vpc, error) {
	request := vpc.CreateDescribeVpcsRequest()
	request.VpcId = vpcID
	response, err := c.DescribeVpcs(request)
	if err != nil {
		return nil, err
	}

	return response.Vpcs.Vpc, nil
}

// GetNatGatewaysWithVPCID returns Gateways with given vpcID.
func (c *vpcClient) GetNatGatewaysWithVPCID(ctx context.Context, vpcID string) ([]vpc.NatGateway, error) {
	request := vpc.CreateDescribeNatGatewaysRequest()
	request.VpcId = vpcID
	response, err := c.DescribeNatGateways(request)
	if err != nil {
		return nil, err
	}

	return response.NatGateways.NatGateway, nil
}

// GetEIPWithID returns EIP with given eipID
func (c *vpcClient) GetEIPWithID(ctx context.Context, eipID string) ([]vpc.EipAddress, error) {
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
