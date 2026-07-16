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
		hasBYO      bool
		hasManaged  bool
	)

	for i, zone := range infra.Networks.Zones {
		zonePath := networksPath.Child("zones").Index(i)
		hasWorkersCIDR := zone.Workers != "" || zone.Worker != ""
		hasWorkersVSwitchID := zone.WorkersVSwitchID != nil

		// XOR: exactly one of Workers CIDR or WorkersVSwitchID must be set
		if !hasWorkersCIDR && !hasWorkersVSwitchID {
			allErrs = append(allErrs, field.Required(zonePath,
				"must specify either workers (CIDR) or workersVSwitchID"))
		} else if hasWorkersCIDR && hasWorkersVSwitchID {
			allErrs = append(allErrs, field.Invalid(zonePath, nil,
				"workers CIDR and workersVSwitchID are mutually exclusive; specify only one"))
		}

		if hasWorkersVSwitchID {
			hasBYO = true
			// natGateway is incompatible with BYO VSwitch
			if zone.NatGateway != nil {
				allErrs = append(allErrs, field.Forbidden(zonePath.Child("natGateway"),
					"natGateway cannot be set when workersVSwitchID is used"))
			}
		} else {
			hasManaged = true
			// CIDR validations for Gardener-managed zones
			if zone.Worker != "" {
				workerPath := zonePath.Child("worker")
				cidrs = append(cidrs, cidrvalidation.NewCIDR(zone.Worker, workerPath))
				allErrs = append(allErrs, cidrvalidation.ValidateCIDRIsCanonical(workerPath, zone.Worker)...)
				workerCIDRs = append(workerCIDRs, cidrvalidation.NewCIDR(zone.Worker, workerPath))
			}
			if zone.Workers != "" {
				workerPath := zonePath.Child("workers")
				cidrs = append(cidrs, cidrvalidation.NewCIDR(zone.Workers, workerPath))
				allErrs = append(allErrs, cidrvalidation.ValidateCIDRIsCanonical(workerPath, zone.Workers)...)
				workerCIDRs = append(workerCIDRs, cidrvalidation.NewCIDR(zone.Workers, workerPath))
			}
			allErrs = append(allErrs, ValidateNatGatewayConfig(zone.NatGateway, zonePath.Child("natGateway"))...)
		}
	}

	// All zones must use the same mode
	if hasBYO && hasManaged {
		allErrs = append(allErrs, field.Forbidden(networksPath.Child("zones"),
			"all zones must use the same approach: either all workersVSwitchID (BYO) or all workers CIDR (Gardener-managed); mixing is not allowed"))
	}
	// isBYOMode is true only when ALL zones are BYO (hasBYO=true, hasManaged=false).
	// When mixed (both true), the Forbidden error above is the root cause — avoid cascading errors.
	isBYOMode := hasBYO && !hasManaged

	// BYO VSwitch requires vpc.id and forbids incompatible VPC fields
	if isBYOMode {
		if infra.Networks.VPC.ID == nil {
			allErrs = append(allErrs, field.Required(networksPath.Child("vpc", "id"),
				"vpc.id is required when workersVSwitchID is set"))
		}
		if infra.Networks.VPC.GardenerManagedNATGateway != nil {
			allErrs = append(allErrs, field.Forbidden(networksPath.Child("vpc", "gardenerManagedNATGateway"),
				"gardenerManagedNATGateway cannot be set when workersVSwitchID is used"))
		}
		if infra.Networks.VPC.UseCustomRouteTable != nil {
			allErrs = append(allErrs, field.Forbidden(networksPath.Child("vpc", "useCustomRouteTable"),
				"useCustomRouteTable cannot be set when workersVSwitchID is used"))
		}
	}

	// nodesSecurityGroupID requires vpc.id (independent of BYO VSwitch mode)
	if infra.Networks.NodesSecurityGroupID != nil && infra.Networks.VPC.ID == nil {
		allErrs = append(allErrs, field.Required(networksPath.Child("vpc", "id"),
			"vpc.id is required when nodesSecurityGroupID is set"))
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

	// When useCustomRouteTable is enabled with a user-provided VPC, gardenerManagedNATGateway must be true.
	// This ensures each shoot manages its own NAT Gateway, preventing a multi-shoot VPC scenario where
	// one shoot's cleanup deletes a shared NAT Gateway that other shoots in the same VPC depend on.
	// Only applies when Gardener manages all subnets (no BYO zones).
	if !hasBYO &&
		infra.Networks.VPC.ID != nil &&
		infra.Networks.VPC.UseCustomRouteTable != nil && *infra.Networks.VPC.UseCustomRouteTable &&
		(infra.Networks.VPC.GardenerManagedNATGateway == nil || !*infra.Networks.VPC.GardenerManagedNATGateway) {
		allErrs = append(allErrs, field.Required(
			networksPath.Child("vpc", "gardenerManagedNATGateway"),
			"gardenerManagedNATGateway must be true when useCustomRouteTable is enabled with a user-provided VPC",
		))
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
		// NLB region check applies to both VPC types
		if !nlbSupportedRegions.Has(region) {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("region"), region,
				fmt.Sprintf("region %s does not support NLB which is required for dualStack; supported regions: %s",
					region, strings.Join(sets.List(nlbSupportedRegions), ", "))))
		}

		isUserVPC := infra.Networks.VPC.ID != nil
		seen := sets.New[int]()
		for i, zone := range infra.Networks.Zones {
			zonePath := networksPath.Child("zones").Index(i).Child("ipv6CidrBlock")

			// In BYO VSwitch mode, ipv6CidrBlock is optional: user pre-configures IPv6 on the VSwitch.
			// Skip both the Required check and the uniqueness check for BYO zones.
			if zone.WorkersVSwitchID != nil {
				continue
			}

			var effective int
			if zone.Ipv6CidrBlock == nil {
				if isUserVPC {
					allErrs = append(allErrs, field.Required(zonePath,
						"ipv6CidrBlock is required when dualStack.enabled is true and using a user-provided VPC"))
					continue
				}
				// Gardener-managed VPC: default to the zone's array index
				effective = i
			} else {
				effective = *zone.Ipv6CidrBlock
				if effective < 0 || effective > 255 {
					allErrs = append(allErrs, field.Invalid(zonePath, effective,
						"ipv6CidrBlock must be in range 0-255"))
					continue
				}
			}
			if seen.Has(effective) {
				if zone.Ipv6CidrBlock == nil {
					allErrs = append(allErrs, field.Invalid(zonePath, effective,
						fmt.Sprintf("default ipv6CidrBlock (zone index %d) conflicts with another zone; set ipv6CidrBlock explicitly to resolve", effective)))
				} else {
					allErrs = append(allErrs, field.Invalid(zonePath, effective,
						"ipv6CidrBlock must be unique across zones"))
				}
			} else {
				seen.Insert(effective)
			}
		}
	}

	return allErrs
}

// ValidateInfrastructureConfigUpdate validates a InfrastructureConfig object.
func ValidateInfrastructureConfigUpdate(oldConfig, newConfig *apisalicloud.InfrastructureConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	networksPath := field.NewPath("networks")
	vpcPath := networksPath.Child("vpc")

	// UseCustomRouteTable can only be specified at shoot creation time; any change after creation is forbidden.
	// This includes both enabling (nil/false → true) and disabling (true → false/nil).
	// nil and false are treated as equivalent (both mean "disabled"), so a nil↔false no-op is permitted.
	// Strip UseCustomRouteTable from both sides before the whole-struct comparison so that the
	// general immutability check does not fire on it, then enforce the creation-time-only rule separately.
	normalizedOldVPC := oldConfig.Networks.VPC
	normalizedNewVPC := newConfig.Networks.VPC
	normalizedOldVPC.UseCustomRouteTable = nil
	normalizedNewVPC.UseCustomRouteTable = nil
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(normalizedNewVPC, normalizedOldVPC, vpcPath)...)

	// Any change in effective value (nil/false ↔ true in either direction) is forbidden after creation.
	if normalizeUseCustomRouteTable(oldConfig.Networks.VPC.UseCustomRouteTable) !=
		normalizeUseCustomRouteTable(newConfig.Networks.VPC.UseCustomRouteTable) {
		allErrs = append(allErrs, field.Forbidden(
			vpcPath.Child("useCustomRouteTable"),
			"useCustomRouteTable can only be set at shoot creation time and cannot be changed afterwards",
		))
	}

	// nodesSecurityGroupID is fully immutable: value must be identical to creation time (nil stays nil, set value stays same)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(
		newConfig.Networks.NodesSecurityGroupID,
		oldConfig.Networks.NodesSecurityGroupID,
		networksPath.Child("nodesSecurityGroupID"),
	)...)

	allErrs = append(allErrs, ValidateNetworkZonesConfig(newConfig.Networks.Zones, oldConfig.Networks.Zones, networksPath.Child("zones"))...)

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

		oldIsBYO := oldZones[i].WorkersVSwitchID != nil
		newIsBYO := newZones[i].WorkersVSwitchID != nil

		// switching between BYO and CIDR mode is forbidden
		if oldIsBYO != newIsBYO {
			allErrs = append(allErrs, field.Forbidden(fldPath.Index(i),
				"cannot switch between workersVSwitchID (BYO) and workers CIDR (Gardener-managed) after creation"))
		}

		// workersVSwitchID is immutable (symmetric with workers CIDR check below)
		if oldIsBYO {
			allErrs = append(allErrs, apivalidation.ValidateImmutableField(
				newZones[i].WorkersVSwitchID,
				oldZones[i].WorkersVSwitchID,
				fldPath.Index(i).Child("workersVSwitchID"),
			)...)
		}

		if !oldIsBYO {
			if isZoneMigratWorkerToWorkers(oldZones[i], newZones[i]) {
				allErrs = append(allErrs, apivalidation.ValidateImmutableField(oldZones[i].Worker, newZones[i].Workers, fldPath.Index(i))...)
			} else {
				allErrs = append(allErrs, apivalidation.ValidateImmutableField(oldZones[i].Workers, newZones[i].Workers, fldPath.Index(i))...)
				allErrs = append(allErrs, apivalidation.ValidateImmutableField(oldZones[i].Worker, newZones[i].Worker, fldPath.Index(i))...)
			}
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
		if zone.WorkersVSwitchID == nil {
			allErrs = append(allErrs, ValidateNatGatewayConfig(zone.NatGateway, fldPath.Index(i).Child("natGateway"))...)
		}
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
