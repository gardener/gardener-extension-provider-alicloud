// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
)

// configValidator implements ConfigValidator for alicloud infrastructure resources.
type configValidator struct {
	factory aliclient.Factory
	mgr     manager.Manager
	logger  logr.Logger
}

// NewConfigValidator creates a new ConfigValidator.
func NewConfigValidator(mgr manager.Manager, logger logr.Logger, factory aliclient.Factory) infrastructure.ConfigValidator {
	return &configValidator{
		factory: factory,
		mgr:     mgr,
		logger:  logger.WithName("alicloud-infrastructure-config-validator"),
	}
}

// Validate validates the provider config of the given infrastructure resource with the cloud provider.
func (c *configValidator) Validate(ctx context.Context, infra *extensionsv1alpha1.Infrastructure) field.ErrorList {
	allErrs := field.ErrorList{}

	logger := c.logger.WithValues("infrastructure", client.ObjectKeyFromObject(infra))

	config, err := helper.InfrastructureConfigFromInfrastructure(infra)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, err))
		return allErrs
	}

	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, c.mgr.GetClient(), &infra.Spec.SecretRef)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, fmt.Errorf("could not get Alicloud credentials: %+v", err)))
		return allErrs
	}
	actor, err := c.factory.NewActor(credentials.AccessKeyID, credentials.AccessKeySecret, infra.Spec.Region)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, fmt.Errorf("create aliclient actor failed: %+v", err)))
		return allErrs
	}

	isBYOMode := len(config.Networks.Zones) > 0 && config.Networks.Zones[0].WorkersVSwitchID != nil

	// Validate infrastructure config
	createManagedNATGateway := true
	if config.Networks.VPC.ID != nil {
		logger.Info("Validating infrastructure networks.vpc.id")
		if config.Networks.VPC.GardenerManagedNATGateway == nil || !*config.Networks.VPC.GardenerManagedNATGateway {
			createManagedNATGateway = false
		}

		// In BYO mode, skip NAT Gateway existence check: user manages routing
		if !isBYOMode {
			allErrs = append(allErrs, c.validateVPC(ctx, actor, *config.Networks.VPC.ID, !createManagedNATGateway, field.NewPath("networks", "vpc", "id"))...)
		} else {
			allErrs = append(allErrs, c.validateVPC(ctx, actor, *config.Networks.VPC.ID, false, field.NewPath("networks", "vpc", "id"))...)
		}

		if config.DualStack != nil && config.DualStack.Enabled {
			logger.Info("Validating VPC IPv6 support for dualStack")
			allErrs = append(allErrs, c.validateVpcIPv6(ctx, actor, *config.Networks.VPC.ID, field.NewPath("networks", "vpc", "id"))...)
		}

		if infra.Status.LastOperation != nil && infra.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeCreate {
			vswitches, err := actor.FindVSwitchesByVPC(ctx, *config.Networks.VPC.ID)
			if err != nil {
				allErrs = append(allErrs, field.InternalError(field.NewPath("networks", "vpc", "id"),
					fmt.Errorf("FindVSwitchesByVPC %s failed: %+v", *config.Networks.VPC.ID, err)))
			} else {
				if !isBYOMode {
					logger.Info("Validating multi-shoot VPC sharing constraints for new shoot")
					allErrs = append(allErrs, c.validateMultiShootVPC(vswitches, infra.Namespace, config.Networks.VPC.GardenerManagedNATGateway, config.Networks.VPC.UseCustomRouteTable, field.NewPath("networks", "vpc"))...)

					logger.Info("Validating vswitch CIDR conflicts for new shoot")
					allErrs = append(allErrs, c.validateVSwitchCIDRConflict(vswitches, *config.Networks.VPC.ID, infra.Namespace, config.Networks.Zones)...)
				}

				// BYO VSwitch validations (Create only)
				if isBYOMode {
					logger.Info("Validating BYO vswitch IDs")
					allErrs = append(allErrs, c.validateBYOVSwitches(ctx, actor, config, *config.Networks.VPC.ID, field.NewPath("networks", "zones"))...)
				}

				// nodesSecurityGroupID validation (Create only, both BYO and Managed modes)
				if config.Networks.NodesSecurityGroupID != nil {
					logger.Info("Validating nodesSecurityGroupID")
					allErrs = append(allErrs, c.validateNodesSecurityGroup(ctx, actor, *config.Networks.NodesSecurityGroupID, *config.Networks.VPC.ID, field.NewPath("networks", "nodesSecurityGroupID"))...)
				}
			}
		}
	}

	if !isBYOMode && createManagedNATGateway {
		logger.Info("Validating infrastructure networks.zones[0].name")
		allErrs = append(allErrs, c.validateEnhancedNatGatewayZone(ctx, actor, config.Networks.Zones[0].Name, infra.Spec.Region, field.NewPath("networks", "zones[0]", "name"))...)
	}

	if !isBYOMode {
		eipIds := sets.New[string]()
		for _, zone := range config.Networks.Zones {
			if zone.NatGateway != nil && zone.NatGateway.EIPAllocationID != nil {
				logger.Info("Validating infrastructure networks.zones[].natGateway.eipAllocationID")
				fldPath := field.NewPath("networks", "zones[]", "natGateway", "eipAllocationID")
				eipId := *zone.NatGateway.EIPAllocationID
				if !eipIds.Has(eipId) {
					eipIds.Insert(eipId)
					allErrs = append(allErrs, c.validateEIP(ctx, actor, eipId, fldPath)...)
				} else {
					allErrs = append(allErrs, field.Forbidden(fldPath, fmt.Sprintf("Duplicate EIP Allocation ID %s", eipId)))
				}
			}
		}
	}

	return allErrs
}

func (c *configValidator) validateVPC(ctx context.Context, actor aliclient.Actor, vpcID string, checkNatgatewayExists bool, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// check vpc if exists
	vpc, err := actor.GetVpc(ctx, vpcID)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(fldPath, fmt.Errorf("validateVPC GetVpc %s failed: %+v", vpcID, err)))
		return allErrs
	}
	if vpc == nil {
		allErrs = append(allErrs, field.NotFound(fldPath, vpcID))
		return allErrs
	}
	if checkNatgatewayExists {
		gw_list, err := actor.ListNatGatewaysByVPC(ctx, vpcID)
		if err != nil {
			allErrs = append(allErrs, field.InternalError(fldPath, fmt.Errorf("validateVPC FindNatGatewayByVPC %s failed: %+v", vpcID, err)))
			return allErrs
		}
		// DescribeNatGateways does not return tag data, so fetch tags separately to identify
		// NAT Gateways managed by other Gardener shoots (tagged kubernetes.io/cluster/<namespace>).
		var gwIds []string
		for _, gw := range gw_list {
			gwIds = append(gwIds, gw.NatGatewayId)
		}
		gwTags, err := actor.GetNatGatewayTags(ctx, gwIds)
		if err != nil {
			allErrs = append(allErrs, field.InternalError(fldPath, fmt.Errorf("validateVPC GetNatGatewayTags %s failed: %+v", vpcID, err)))
			return allErrs
		}
		var userGwList []*aliclient.NatGateway
		for _, gw := range gw_list {
			tags := gwTags[gw.NatGatewayId]
			if !aliclient.IsGardenerManaged(tags) {
				userGwList = append(userGwList, gw)
			}
		}
		if len(userGwList) == 0 {
			allErrs = append(allErrs, field.Invalid(fldPath, vpcID, "no user natgateway found"))
			return allErrs
		}
		if len(userGwList) > 1 {
			allErrs = append(allErrs, field.Invalid(fldPath, vpcID, "more than one user natgateway found"))
			return allErrs
		}
	}

	return allErrs
}

func (c *configValidator) validateEIP(ctx context.Context, actor aliclient.Actor, eipId string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	eip, err := actor.GetEIP(ctx, eipId)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(fldPath, fmt.Errorf("validateEIP GetEIP %s failed: %+v", eipId, err)))
		return allErrs
	}
	if eip == nil {
		allErrs = append(allErrs, field.NotFound(fldPath, eipId))
	}
	return allErrs
}

func (c *configValidator) validateEnhancedNatGatewayZone(ctx context.Context, actor aliclient.Actor, zone, region string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	validZones, err := actor.ListEnhanhcedNatGatewayAvailableZones(ctx, region)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, fmt.Errorf("list natgateway availableZones failed: %+v", err)))
		return allErrs
	}
	validNatGatewayZone := false
	for _, valid_zone := range validZones {
		if zone == valid_zone {
			validNatGatewayZone = true
			break
		}
	}
	if !validNatGatewayZone {
		allErrs = append(allErrs, field.Forbidden(fldPath, fmt.Sprintf("zone %s does not support enhance natgateway, please use following zones: %s", zone, strings.Join(validZones, " "))))
	}
	return allErrs
}

func (c *configValidator) validateVpcIPv6(ctx context.Context, actor aliclient.Actor, vpcID string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Check whether VPC has IPv6 enabled
	ipv6Cidr, err := actor.GetVpcIpv6Info(ctx, vpcID)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(fldPath, fmt.Errorf("validateVpcIPv6 GetVpcIpv6Info %s failed: %+v", vpcID, err)))
		return allErrs
	}
	if ipv6Cidr == "" {
		allErrs = append(allErrs, field.Invalid(fldPath, vpcID,
			"VPC does not have IPv6 enabled; please enable IPv6 on the VPC before using dualStack"))
		return allErrs
	}

	// Check whether VPC has an IPv6 Gateway
	gw, err := actor.FindIpv6GatewayByVPC(ctx, vpcID)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(fldPath, fmt.Errorf("validateVpcIPv6 FindIpv6GatewayByVPC %s failed: %+v", vpcID, err)))
		return allErrs
	}
	if gw == nil {
		allErrs = append(allErrs, field.Invalid(fldPath, vpcID,
			"VPC does not have an IPv6 Gateway; please create one before using dualStack"))
	}
	return allErrs
}

func (c *configValidator) validateMultiShootVPC(vswitches []*aliclient.VSwitch, namespace string, gardenerManagedNATGateway *bool, useCustomRouteTable *bool, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	ownClusterTagKey := fmt.Sprintf("kubernetes.io/cluster/%s", namespace)
	hasOtherShoot := false
	for _, vsw := range vswitches {
		for k := range vsw.Tags {
			if strings.HasPrefix(k, "kubernetes.io/cluster/") && k != ownClusterTagKey {
				hasOtherShoot = true
				break
			}
		}
		if hasOtherShoot {
			break
		}
	}

	if hasOtherShoot {
		if gardenerManagedNATGateway != nil && *gardenerManagedNATGateway {
			if useCustomRouteTable == nil || !*useCustomRouteTable {
				allErrs = append(allErrs, field.Required(
					fldPath.Child("useCustomRouteTable"),
					"useCustomRouteTable must be true when gardenerManagedNATGateway is true and the VPC already contains other Gardener-managed shoots",
				))
			}
		}
	}

	return allErrs
}

// validateBYOVSwitches validates each workersVSwitchID:
// - must exist in the specified VPC
// - must be in the zone specified by zone.Name
// - if dualStack: VSwitch must have IPv6 CIDR configured
func (c *configValidator) validateBYOVSwitches(ctx context.Context, actor aliclient.Actor, config *apisalicloud.InfrastructureConfig, vpcID string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for i, zone := range config.Networks.Zones {
		if zone.WorkersVSwitchID == nil {
			continue
		}
		zonePath := fldPath.Index(i).Child("workersVSwitchID")
		vswID := *zone.WorkersVSwitchID

		vsw, err := actor.GetVSwitch(ctx, vswID)
		if err != nil {
			allErrs = append(allErrs, field.InternalError(zonePath,
				fmt.Errorf("GetVSwitch %s failed: %+v", vswID, err)))
			continue
		}
		if vsw == nil {
			allErrs = append(allErrs, field.NotFound(zonePath, vswID))
			continue
		}

		// VSwitch must belong to the specified VPC
		if vsw.VpcId == nil || *vsw.VpcId != vpcID {
			allErrs = append(allErrs, field.Invalid(zonePath, vswID,
				fmt.Sprintf("VSwitch does not belong to VPC %s", vpcID)))
		}

		// VSwitch must be in the zone specified by zone.Name
		if vsw.ZoneId != zone.Name {
			allErrs = append(allErrs, field.Invalid(zonePath, vswID,
				fmt.Sprintf("VSwitch is in zone %s but zone.name specifies %s", vsw.ZoneId, zone.Name)))
		}

		// dualStack: VSwitch must have IPv6 CIDR pre-configured
		if config.DualStack != nil && config.DualStack.Enabled {
			ipv6Cidr, err := actor.GetVSwitchIpv6CidrBlock(ctx, vswID)
			if err != nil {
				allErrs = append(allErrs, field.InternalError(zonePath,
					fmt.Errorf("GetVSwitchIpv6CidrBlock %s failed: %+v", vswID, err)))
			} else if ipv6Cidr == "" {
				allErrs = append(allErrs, field.Invalid(zonePath, vswID,
					"VSwitch does not have IPv6 CIDR configured; please enable IPv6 on the VSwitch before using dualStack"))
			}
		}
	}

	return allErrs
}

// validateNodesSecurityGroup validates that the specified security group exists in the given VPC.
func (c *configValidator) validateNodesSecurityGroup(ctx context.Context, actor aliclient.Actor, sgID, vpcID string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	sg, err := actor.GetSecurityGroup(ctx, sgID)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(fldPath,
			fmt.Errorf("GetSecurityGroup %s failed: %+v", sgID, err)))
		return allErrs
	}
	if sg == nil {
		allErrs = append(allErrs, field.NotFound(fldPath, sgID))
		return allErrs
	}
	if sg.VpcId != vpcID {
		allErrs = append(allErrs, field.Invalid(fldPath, sgID,
			fmt.Sprintf("security group does not belong to VPC %s", vpcID)))
	}

	return allErrs
}

// vswitches already existing in the VPC that are not owned by this shoot.
// Called only on create, but create may be retried after partial failure, so vswitches whose name
// starts with "<namespace>-" (the naming convention used by this extension) are excluded to avoid
// false positives on retry.
func (c *configValidator) validateVSwitchCIDRConflict(vswitches []*aliclient.VSwitch, vpcID string, namespace string, zones []apisalicloud.Zone) field.ErrorList {
	allErrs := field.ErrorList{}

	ownPrefix := namespace + "-"
	var foreignVSwitches []*aliclient.VSwitch
	for _, vsw := range vswitches {
		if !strings.HasPrefix(vsw.Name, ownPrefix) {
			foreignVSwitches = append(foreignVSwitches, vsw)
		}
	}

	zonesPath := field.NewPath("networks", "zones")
	for i, zone := range zones {
		workerCIDR := zone.Workers
		if workerCIDR == "" {
			workerCIDR = zone.Worker
		}
		if workerCIDR == "" {
			continue
		}
		fldPath := zonesPath.Index(i).Child("workers")
		_, netA, err := net.ParseCIDR(workerCIDR)
		if err != nil {
			continue
		}
		for _, vsw := range foreignVSwitches {
			_, netB, err := net.ParseCIDR(vsw.CidrBlock)
			if err != nil {
				continue
			}
			if netA.Contains(netB.IP) || netB.Contains(netA.IP) {
				allErrs = append(allErrs, field.Invalid(fldPath, workerCIDR,
					fmt.Sprintf("conflicts with existing vswitch %s (%s) in VPC %s", vsw.VSwitchId, vsw.CidrBlock, vpcID)))
			}
		}
	}
	return allErrs
}
