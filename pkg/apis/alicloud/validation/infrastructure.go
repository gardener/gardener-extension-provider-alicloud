// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package validation

import (
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	cidrvalidation "github.com/gardener/gardener/pkg/utils/validation/cidr"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateInfrastructureConfig validates a InfrastructureConfig object.
func ValidateInfrastructureConfig(infra *apisalicloud.InfrastructureConfig, nodesCIDR, podsCIDR, servicesCIDR *string) field.ErrorList {
	allErrs := field.ErrorList{}

	var (
		nodes    cidrvalidation.CIDR
		pods     cidrvalidation.CIDR
		services cidrvalidation.CIDR
	)

	if nodesCIDR != nil {
		nodes = cidrvalidation.NewCIDR(*nodesCIDR, nil)
	}
	if podsCIDR != nil {
		pods = cidrvalidation.NewCIDR(*podsCIDR, nil)
	}
	if servicesCIDR != nil {
		services = cidrvalidation.NewCIDR(*servicesCIDR, nil)
	}

	networksPath := field.NewPath("networks")
	if len(infra.Networks.Zones) == 0 {
		allErrs = append(allErrs, field.Required(networksPath.Child("zones"), "must specify at least the network for one zone"))
	}

	var (
		cidrs       = make([]cidrvalidation.CIDR, 0, len(infra.Networks.Zones))
		workerCIDRs = make([]cidrvalidation.CIDR, 0, len(infra.Networks.Zones))
	)

	for i, zone := range infra.Networks.Zones {
		if zone.Worker != "" {
			workerPath := networksPath.Child("zones").Index(i).Child("worker")
			cidrs = append(cidrs, cidrvalidation.NewCIDR(zone.Worker, workerPath))
			allErrs = append(allErrs, cidrvalidation.ValidateCIDRIsCanonical(workerPath, zone.Worker)...)
			workerCIDRs = append(workerCIDRs, cidrvalidation.NewCIDR(zone.Worker, workerPath))
		}

		if zone.Workers != "" {
			workerPath := networksPath.Child("zones").Index(i).Child("workers")
			cidrs = append(cidrs, cidrvalidation.NewCIDR(zone.Workers, workerPath))
			allErrs = append(allErrs, cidrvalidation.ValidateCIDRIsCanonical(workerPath, zone.Workers)...)
			workerCIDRs = append(workerCIDRs, cidrvalidation.NewCIDR(zone.Workers, workerPath))
		}

		allErrs = append(allErrs, ValidateNatGatewayConfig(zone.NatGateway, networksPath.Child("zones").Index(i).Child("natGateway"))...)
	}

	allErrs = append(allErrs, cidrvalidation.ValidateCIDRParse(cidrs...)...)

	if nodes != nil {
		allErrs = append(allErrs, nodes.ValidateSubset(workerCIDRs...)...)
	}

	if (infra.Networks.VPC.ID == nil && infra.Networks.VPC.CIDR == nil) || (infra.Networks.VPC.ID != nil && infra.Networks.VPC.CIDR != nil) {
		allErrs = append(allErrs, field.Invalid(networksPath.Child("vpc"), infra.Networks.VPC, "must specify either a vpc id or a cidr"))
	} else if infra.Networks.VPC.CIDR != nil && infra.Networks.VPC.ID == nil {
		cidrPath := networksPath.Child("vpc", "cidr")
		vpcCIDR := cidrvalidation.NewCIDR(*infra.Networks.VPC.CIDR, cidrPath)
		allErrs = append(allErrs, cidrvalidation.ValidateCIDRIsCanonical(cidrPath, *infra.Networks.VPC.CIDR)...)
		allErrs = append(allErrs, vpcCIDR.ValidateParse()...)
		allErrs = append(allErrs, vpcCIDR.ValidateSubset(nodes)...)
		allErrs = append(allErrs, vpcCIDR.ValidateSubset(cidrs...)...)
		allErrs = append(allErrs, vpcCIDR.ValidateNotSubset(pods, services)...)
	}

	// make sure that VPC cidrs don't overlap with each other
	allErrs = append(allErrs, cidrvalidation.ValidateCIDROverlap(cidrs, cidrs, false)...)
	allErrs = append(allErrs, cidrvalidation.ValidateCIDROverlap([]cidrvalidation.CIDR{pods, services}, cidrs, false)...)

	return allErrs
}

// ValidateInfrastructureConfigUpdate validates a InfrastructureConfig object.
func ValidateInfrastructureConfigUpdate(oldConfig, newConfig *apisalicloud.InfrastructureConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apivalidation.ValidateImmutableField(newConfig.Networks.VPC, oldConfig.Networks.VPC, field.NewPath("networks").Child("vpc"))...)
	allErrs = append(allErrs, ValidateNetworkZonesConfig(newConfig.Networks.Zones, oldConfig.Networks.Zones, field.NewPath("networks").Child("zones"))...)

	return allErrs
}

func ValidateNetworkZonesConfig(newZones, oldZones []apisalicloud.Zone, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(newZones) < len(oldZones) {
		allErrs = append(allErrs, field.Forbidden(fldPath, "zones cannot be removed"))
		return allErrs
	}

	for i := range oldZones {
		allErrs = append(allErrs, apivalidation.ValidateImmutableField(oldZones[i].Name, newZones[i].Name, fldPath.Index(i))...)
		allErrs = append(allErrs, apivalidation.ValidateImmutableField(oldZones[i].Workers, newZones[i].Workers, fldPath.Index(i))...)
		allErrs = append(allErrs, apivalidation.ValidateImmutableField(oldZones[i].Worker, newZones[i].Worker, fldPath.Index(i))...)
	}

	for i, zone := range newZones {
		allErrs = append(allErrs, ValidateNatGatewayConfig(zone.NatGateway, fldPath.Index(i).Child("natGateway"))...)
	}

	return allErrs
}

// ValidateNatGatewayConfig validates a NatGatewayConfig object.
func ValidateNatGatewayConfig(natGateway *apisalicloud.NatGatewayConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if natGateway != nil {
		if natGateway.EIPAllocationID == nil {
			allErrs = append(allErrs, field.Invalid(fldPath, natGateway, "eip id is not specified"))
		} else if *natGateway.EIPAllocationID == "" {
			allErrs = append(allErrs, field.Invalid(fldPath, natGateway, "eip id cannot be empty string"))
		}
	}

	return allErrs
}
