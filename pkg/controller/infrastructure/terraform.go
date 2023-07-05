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
	"embed"
	"strconv"
	"text/template"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
)

var (
	//go:embed templates/main.tpl.tf
	//go:embed templates/terraform.tpl.tfvars
	//go:embed templates/variables.tpl.tf
	tplsTF          embed.FS
	variablesTF     []byte
	terraformTFVars []byte
	tplMainTF       *template.Template
)

func init() {
	var (
		err           error
		tplNameMainTF = "main.tf"
	)

	mainScriptTF, err := tplsTF.ReadFile("templates/main.tpl.tf")
	if err != nil {
		panic(err)
	}
	tplMainTF, err = template.
		New(tplNameMainTF).
		Parse(string(mainScriptTF))
	if err != nil {
		panic(err)
	}

	variablesTF, err = tplsTF.ReadFile("templates/variables.tpl.tf")
	if err != nil {
		panic(err)
	}

	terraformTFVars, err = tplsTF.ReadFile("templates/terraform.tpl.tfvars")
	if err != nil {
		panic(err)
	}
}

type terraformOps struct{}

// DefaultTerraformOps returns the default TerraformChartOps.
func DefaultTerraformOps() TerraformChartOps {
	return terraformOps{}
}

// ComputeCreateVPCInitializerValues computes the InitializerValues to create a new VPC.
func (terraformOps) ComputeCreateVPCInitializerValues(config *v1alpha1.InfrastructureConfig, internetChargeType string) *InitializerValues {
	return &InitializerValues{
		VPC: VPC{
			CreateVPC: true,
			VPCID:     TerraformDefaultVPCID,
			VPCCIDR:   *config.Networks.VPC.CIDR,
		},
		NATGateway: NATGateway{
			CreateNATGateway: true,
			NATGatewayID:     TerraformDefaultNATGatewayID,
			SNATTableIDs:     TerraformDefaultSNATTableIDs,
		},
		EIP: EIP{
			InternetChargeType: internetChargeType,
		},
	}
}

// ComputeUseVPCInitializerValues computes the InitializerValues to use an existing VPC.
func (terraformOps) ComputeUseVPCInitializerValues(config *v1alpha1.InfrastructureConfig, info *alicloudclient.VPCInfo) *InitializerValues {
	values := &InitializerValues{
		VPC: VPC{
			CreateVPC: false,
			VPCID:     strconv.Quote(*config.Networks.VPC.ID),
			VPCCIDR:   info.CIDR,
		},
		NATGateway: NATGateway{
			CreateNATGateway: false,
			NATGatewayID:     strconv.Quote(info.NATGatewayID),
			SNATTableIDs:     strconv.Quote(info.SNATTableIDs),
		},
		EIP: EIP{
			InternetChargeType: info.InternetChargeType,
		},
	}
	if config.Networks.VPC.GardenerManagedNATGateway != nil && *config.Networks.VPC.GardenerManagedNATGateway {
		values.NATGateway.CreateNATGateway = true
		values.NATGateway.NATGatewayID = TerraformDefaultNATGatewayID
		values.NATGateway.SNATTableIDs = TerraformDefaultSNATTableIDs
	}

	return values
}

// ComputeTerraformerChartValues computes the values necessary for the infrastructure Terraform chart.
func (terraformOps) ComputeChartValues(
	infra *extensionsv1alpha1.Infrastructure,
	config *v1alpha1.InfrastructureConfig,
	podCIDR *string,
	values *InitializerValues,
) map[string]interface{} {
	zones := make([]map[string]interface{}, 0, len(config.Networks.Zones))
	for _, zone := range config.Networks.Zones {
		workersCIDR := zone.Workers
		// Backwards compatibility - remove this code in a future version.
		if workersCIDR == "" {
			workersCIDR = zone.Worker
		}

		zoneConfig := map[string]interface{}{
			"name": zone.Name,
			"cidr": map[string]interface{}{
				"workers": string(workersCIDR),
			},
		}

		if zone.NatGateway != nil && zone.NatGateway.EIPAllocationID != nil && *zone.NatGateway.EIPAllocationID != "" {
			zoneConfig["eipAllocationID"] = *zone.NatGateway.EIPAllocationID
		}

		zones = append(zones, zoneConfig)
	}
	bandwidth := "100"
	if config.Networks.VPC.Bandwidth != nil && *config.Networks.VPC.Bandwidth != "" {
		bandwidth = *config.Networks.VPC.Bandwidth
	}

	return map[string]interface{}{
		"alicloud": map[string]interface{}{
			"region": infra.Spec.Region,
		},
		"vpc": map[string]interface{}{
			"create": values.VPC.CreateVPC,
			"id":     values.VPC.VPCID,
			"cidr":   values.VPC.VPCCIDR,
		},
		"natGateway": map[string]interface{}{
			"create":       values.NATGateway.CreateNATGateway,
			"id":           values.NATGateway.NATGatewayID,
			"sNatTableIDs": values.NATGateway.SNATTableIDs,
		},
		"eip": map[string]interface{}{
			"internetChargeType": values.EIP.InternetChargeType,
			"bandwidth":          bandwidth,
		},
		"clusterName": infra.Namespace,
		"zones":       zones,
		"podCIDR":     *podCIDR,
		"outputKeys": map[string]interface{}{
			"vpcID":              TerraformerOutputKeyVPCID,
			"vpcCIDR":            TerraformerOutputKeyVPCCIDR,
			"securityGroupID":    TerraformerOutputKeySecurityGroupID,
			"vswitchNodesPrefix": TerraformerOutputKeyVSwitchNodesPrefix,
		},
	}
}
