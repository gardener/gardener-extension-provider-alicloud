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

package infrastructure

import (
	"fmt"
	"strings"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
)

// GetVPCInfo gets info of an existing VPC.
func GetVPCInfo(vpcClient alicloudclient.VPC, vpcID string) (*VPCInfo, error) {
	describeVPCsReq := vpc.CreateDescribeVpcsRequest()
	describeVPCsReq.VpcId = vpcID
	describeVPCsRes, err := vpcClient.DescribeVpcs(describeVPCsReq)
	if err != nil {
		return nil, err
	}

	if len(describeVPCsRes.Vpcs.Vpc) != 1 {
		return nil, fmt.Errorf("ambiguous VPC response: expected 1 VPC but got %v", describeVPCsRes.Vpcs.Vpc)
	}

	vpcCIDR := describeVPCsRes.Vpcs.Vpc[0].CidrBlock

	describeNATGatewaysReq := vpc.CreateDescribeNatGatewaysRequest()
	describeNATGatewaysReq.VpcId = vpcID
	describeNatGatewaysRes, err := vpcClient.DescribeNatGateways(describeNATGatewaysReq)
	if err != nil {
		return nil, err
	}

	if len(describeNatGatewaysRes.NatGateways.NatGateway) != 1 {
		return nil, fmt.Errorf("ambiguous NAT Gateway response: expected 1 NAT Gateway but got %v", describeNatGatewaysRes.NatGateways.NatGateway)
	}

	natGateway := describeNatGatewaysRes.NatGateways.NatGateway[0]
	natGatewayID := natGateway.NatGatewayId
	sNatTableIDs := strings.Join(natGateway.SnatTableIds.SnatTableId, ",")

	internetChargeType, err := FetchEIPInternetChargeType(vpcClient, vpcID)
	if err != nil {
		return nil, err
	}

	return &VPCInfo{
		CIDR:               vpcCIDR,
		NATGatewayID:       natGatewayID,
		SNATTableIDs:       sNatTableIDs,
		InternetChargeType: internetChargeType,
	}, nil
}

// FetchEIPInternetChargeType fetches the internet charge type for the VPC's EIP.
func FetchEIPInternetChargeType(vpcClient alicloudclient.VPC, vpcID string) (string, error) {
	describeNATGatewaysReq := vpc.CreateDescribeNatGatewaysRequest()
	describeNATGatewaysReq.VpcId = vpcID

	describeNatGatewaysRes, err := vpcClient.DescribeNatGateways(describeNATGatewaysReq)
	if err != nil {
		return "", err
	}

	if len(describeNatGatewaysRes.NatGateways.NatGateway) != 1 {
		return alicloudclient.DefaultInternetChargeType, nil
	}

	natGateway := describeNatGatewaysRes.NatGateways.NatGateway[0]

	if len(natGateway.IpLists.IpList) == 0 {
		return alicloudclient.DefaultInternetChargeType, nil
	}

	ipList := natGateway.IpLists.IpList[0]
	eipReq := vpc.CreateDescribeEipAddressesRequest()
	eipReq.AllocationId = ipList.AllocationId

	eipResp, err := vpcClient.DescribeEipAddresses(eipReq)
	if err != nil {
		return "", err
	}

	if len(eipResp.EipAddresses.EipAddress) == 0 {
		return alicloudclient.DefaultInternetChargeType, nil
	}

	return eipResp.EipAddresses.EipAddress[0].InternetChargeType, nil
}
