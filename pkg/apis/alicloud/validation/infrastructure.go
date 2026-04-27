// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"strings"

	"github.com/gardener/gardener/pkg/apis/core"
	cidrvalidation "github.com/gardener/gardener/pkg/utils/validation/cidr"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
)

// nlbSupportedRegions lists Alicloud regions that support NLB (last updated: 2026-04).
// Keep in sync with https://help.aliyun.com/zh/slb/network-load-balancer/product-overview/regions-that-support-nlb
var nlbSupportedRegions = sets.New[string](
	"cn-hangzhou",
	"cn-beijing",
	"cn-shenzhen",
	"cn-shanghai",
	"cn-qingdao",
	"cn-zhangjiakou",
	"cn-chengdu",
	"cn-guangzhou",
	"cn-hongkong",
	"cn-heyuan",
	"cn-wulanchabu",
	"ap-southeast-7",
	"ap-southeast-6",
	"ap-southeast-1",
	"ap-northeast-1",
	"ap-northeast-2",
	"ap-southeast-3",
	"ap-southeast-5",
	"eu-central-1",
	"eu-west-1",
	"us-east-1",
	"us-west-1",
	"na-south-1",
)

// ValidateInfrastructureConfig validates a InfrastructureConfig object.
func ValidateInfrastructureConfig(infra *apisalicloud.InfrastructureConfig, networking *core.Networking, region string) field.ErrorList {
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

	// make sure that VPC cidrs don't overlap with each other
	allErrs = append(allErrs, cidrvalidation.ValidateCIDROverlap(cidrs, false)...)
	if pods != nil {
		allErrs = append(allErrs, pods.ValidateNotOverlap(cidrs...)...)
	}
	if services != nil {
		allErrs = append(allErrs, services.ValidateNotOverlap(cidrs...)...)
	}

	// DualStack validation
	if infra.DualStack != nil && infra.DualStack.Enabled {
		isManagedVPC := infra.Networks.VPC.ID == nil
		seen := sets.New[int]()
		for i, zone := range infra.Networks.Zones {
			zonePath := networksPath.Child("zones").Index(i).Child("ipv6CidrBlock")
			if zone.Ipv6CidrBlock == nil {
				if isManagedVPC {
					allErrs = append(allErrs, field.Required(zonePath,
						"ipv6CidrBlock is required when dualStack.enabled is true and VPC is managed by Gardener"))
				}
			} else {
				v := *zone.Ipv6CidrBlock
				if v < 0 || v > 255 {
					allErrs = append(allErrs, field.Invalid(zonePath, v,
						"ipv6CidrBlock must be in range 0-255"))
				} else if seen.Has(v) {
					allErrs = append(allErrs, field.Invalid(zonePath, v,
						"ipv6CidrBlock must be unique across zones"))
				} else {
					seen.Insert(v)
				}
			}
		}

		// NLB region check
		if !nlbSupportedRegions.Has(region) {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("region"), region,
				fmt.Sprintf("region %s does not support NLB which is required for dualStack; supported regions: %s",
					region, strings.Join(sets.List(nlbSupportedRegions), ", "))))
		}
	}

	return allErrs
}

// ValidateInfrastructureConfigUpdate validates a InfrastructureConfig object.
func ValidateInfrastructureConfigUpdate(oldConfig, newConfig *apisalicloud.InfrastructureConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	vpcPath := field.NewPath("networks").Child("vpc")
	// UseCustomRouteTable is a one-way switch: nil/false → true is allowed, but true → false/nil is forbidden.
	// Strip UseCustomRouteTable from both sides before the whole-struct comparison so that the
	// general immutability check does not fire on it, then enforce the one-way rule separately.
	normalizedOldVPC := oldConfig.Networks.VPC
	normalizedNewVPC := newConfig.Networks.VPC
	normalizedOldVPC.UseCustomRouteTable = nil
	normalizedNewVPC.UseCustomRouteTable = nil
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(normalizedNewVPC, normalizedOldVPC, vpcPath)...)

	// UseCustomRouteTable is immutable once set: any change between true and false/nil is forbidden.
	// nil and false are treated as equivalent so that a nil→false no-op does not trigger an error.
	if normalizeUseCustomRouteTable(oldConfig.Networks.VPC.UseCustomRouteTable) !=
		normalizeUseCustomRouteTable(newConfig.Networks.VPC.UseCustomRouteTable) {
		allErrs = append(allErrs, field.Forbidden(
			vpcPath.Child("useCustomRouteTable"),
			"useCustomRouteTable is immutable once set",
		))
	}

	allErrs = append(allErrs, ValidateNetworkZonesConfig(newConfig.Networks.Zones, oldConfig.Networks.Zones, field.NewPath("networks").Child("zones"))...)

	// DualStack.Enabled can be enabled but not disabled once set
	oldEnabled := oldConfig.DualStack != nil && oldConfig.DualStack.Enabled
	newEnabled := newConfig.DualStack != nil && newConfig.DualStack.Enabled
	if oldEnabled && !newEnabled {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("dualStack"),
			"dualStack cannot be disabled once enabled"))
	}

	return allErrs
}

// normalizeUseCustomRouteTable treats nil and false as equivalent (both mean "disabled").
func normalizeUseCustomRouteTable(v *bool) bool {
	return v != nil && *v
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
		if isZoneMigratWorkerToWorkers(oldZones[i], newZones[i]) {
			allErrs = append(allErrs, apivalidation.ValidateImmutableField(oldZones[i].Worker, newZones[i].Workers, fldPath.Index(i))...)
		} else {
			allErrs = append(allErrs, apivalidation.ValidateImmutableField(oldZones[i].Workers, newZones[i].Workers, fldPath.Index(i))...)
			allErrs = append(allErrs, apivalidation.ValidateImmutableField(oldZones[i].Worker, newZones[i].Worker, fldPath.Index(i))...)
		}
		// Ipv6CidrBlock can be changed but not removed once set
		if oldZones[i].Ipv6CidrBlock != nil && newZones[i].Ipv6CidrBlock == nil {
			allErrs = append(allErrs, field.Invalid(
				fldPath.Index(i).Child("ipv6CidrBlock"), nil,
				"ipv6CidrBlock cannot be removed once set",
			))
		}
	}

	for i, zone := range newZones {
		allErrs = append(allErrs, ValidateNatGatewayConfig(zone.NatGateway, fldPath.Index(i).Child("natGateway"))...)
	}

	return allErrs
}

// check if migrate from worker to workers
func isZoneMigratWorkerToWorkers(oldZone, newZone apisalicloud.Zone) bool {
	if oldZone.Worker != "" && oldZone.Workers == "" && newZone.Worker == "" && newZone.Workers != "" {
		return true
	}
	return false
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
