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
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
)

const (
	// TerraformerPurpose is the Terraformer purpose for infrastructure operations.
	TerraformerPurpose = "infra"

	// TerraformerOutputKeyVPCID is the output key of the VPC ID.
	TerraformerOutputKeyVPCID = "vpc_id"
	// TerraformerOutputKeyVPCCIDR is the output key of the VPC CIDR.
	TerraformerOutputKeyVPCCIDR = "vpc_cidr"
	// TerraformerOutputKeySecurityGroupID is the output key of the security group ID.
	TerraformerOutputKeySecurityGroupID = "sg_id"
	// TerraformerOutputKeyVSwitchNodesPrefix is the prefix for the vswitches.
	TerraformerOutputKeyVSwitchNodesPrefix = "vswitch_id_z"

	// TerraformDefaultVPCID is the default value for the VPC ID in the chart.
	TerraformDefaultVPCID = "alicloud_vpc.vpc.id"
	// TerraformDefaultNATGatewayID is the default value for the NAT gateway ID in the chart.
	TerraformDefaultNATGatewayID = "alicloud_nat_gateway.nat_gateway.id"
	// TerraformDefaultSNATTableIDs is the default value for the SNAT table IDs in the chart.
	TerraformDefaultSNATTableIDs = "alicloud_nat_gateway.nat_gateway.snat_table_ids"
)

// VPC contains values of VPC used to render terraform charts.
type VPC struct {
	CreateVPC bool
	VPCID     string
	VPCCIDR   string
}

// NATGateway contains values of NATGateway used to render terraform charts.
type NATGateway struct {
	NATGatewayID string
	SNATTableIDs string
}

// EIP contains values of EIP used to render terraform charts
type EIP struct {
	InternetChargeType string
}

// InitializerValues are values used to render a terraform initializer chart.
type InitializerValues struct {
	VPC        VPC
	NATGateway NATGateway
	EIP        EIP
}

// TerraformChartOps are operations to do for interfacing with Terraform charts.
type TerraformChartOps interface {
	ComputeCreateVPCInitializerValues(config *v1alpha1.InfrastructureConfig, internetChargeType string) *InitializerValues
	ComputeUseVPCInitializerValues(config *v1alpha1.InfrastructureConfig, info *alicloudclient.VPCInfo) *InitializerValues
	ComputeChartValues(infra *extensionsv1alpha1.Infrastructure, config *v1alpha1.InfrastructureConfig, values *InitializerValues) map[string]interface{}
}
