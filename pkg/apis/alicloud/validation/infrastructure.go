// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/gardener/gardener/pkg/apis/core"
	cidrvalidation "github.com/gardener/gardener/pkg/utils/validation/cidr"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
)

// ValidateInfrastructureConfig validates a InfrastructureConfig object.
func ValidateInfrastructureConfig(infra *apisalicloud.InfrastructureConfig, networking *core.Networking) field.ErrorList {
	allErrs := field.ErrorList{}

	var (
		nodes, pods, services             cidrvalidation.CIDR
		nodesCIDR, podsCIDR, servicesCIDR *string
	)

	if networking != nil {
		networkingPath := field.NewPath("networking")
		if nodesCIDR = networking.Nodes; nodesCIDR != nil {
			nodes = cidrvalidation.NewCIDR(*nodesCIDR, networkingPath.Child("nodes"))
		}
		if podsCIDR = networking.Pods; podsCIDR != nil {
			pods = cidrvalidation.NewCIDR(*podsCIDR, networkingPath.Child("pods"))
		}
		if servicesCIDR = networking.Services; servicesCIDR != nil {
			services = cidrvalidation.NewCIDR(*servicesCIDR, networkingPath.Child("services"))
		}
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
		allErrs = append(allErrs, vpcCIDR.ValidateNotOverlap(pods, services)...)
	}
	if infra.DualStack != nil && infra.DualStack.Enabled && infra.Networks.VPC.ID != nil {
		allErrs = append(allErrs, field.Invalid(networksPath.Child("vpc"), infra.Networks.VPC, "can not set vpc id when DualStack enabled"))
	}

	// make sure that VPC cidrs don't overlap with each other
	allErrs = append(allErrs, cidrvalidation.ValidateCIDROverlap(cidrs, false)...)
	if pods != nil {
		allErrs = append(allErrs, pods.ValidateNotOverlap(cidrs...)...)
	}
	if services != nil {
		allErrs = append(allErrs, services.ValidateNotOverlap(cidrs...)...)
	}

	return allErrs
}

// ValidateInfrastructureConfigUpdate validates a InfrastructureConfig object.
func ValidateInfrastructureConfigUpdate(oldConfig, newConfig *apisalicloud.InfrastructureConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apivalidation.ValidateImmutableField(newConfig.Networks.VPC, oldConfig.Networks.VPC, field.NewPath("networks").Child("vpc"))...)
	allErrs = append(allErrs, ValidateNetworkZonesConfig(newConfig.Networks.Zones, oldConfig.Networks.Zones, field.NewPath("networks").Child("zones"))...)

	if oldConfig.DualStack != nil && oldConfig.DualStack.Enabled && (newConfig.DualStack == nil || !newConfig.DualStack.Enabled) {
		dualStackPath := field.NewPath("dualStack.enabled")
		allErrs = append(allErrs, field.Forbidden(dualStackPath, "field can't be changed from \"true\" to \"false\""))
	}

	return allErrs
}

// ValidateNetworkZonesConfig validates a Zone slice.
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
