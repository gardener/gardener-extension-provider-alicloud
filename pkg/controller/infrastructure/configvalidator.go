// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"
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

	// Validate infrastructure config
	createManagedNATGateway := true
	if config.Networks.VPC.ID != nil {
		logger.Info("Validating infrastructure networks.vpc.id")
		if config.Networks.VPC.GardenerManagedNATGateway == nil || !*config.Networks.VPC.GardenerManagedNATGateway {
			createManagedNATGateway = false
		}
		allErrs = append(allErrs, c.validateVPC(ctx, actor, *config.Networks.VPC.ID, !createManagedNATGateway, field.NewPath("networks", "vpc", "id"))...)

		if config.DualStack != nil && config.DualStack.Enabled {
			logger.Info("Validating VPC IPv6 support for dualStack")
			allErrs = append(allErrs, c.validateVpcIPv6(ctx, actor, *config.Networks.VPC.ID, field.NewPath("networks", "vpc", "id"))...)
		}

		if infra.Status.LastOperation != nil && infra.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeCreate {
			logger.Info("Validating multi-shoot VPC sharing constraints for new shoot")
			allErrs = append(allErrs, c.validateMultiShootVPC(ctx, actor, *config.Networks.VPC.ID, infra.Namespace, config.Networks.VPC.GardenerManagedNATGateway, config.Networks.VPC.UseCustomRouteTable, field.NewPath("networks", "vpc"))...)
		}
	}
	if createManagedNATGateway {
		logger.Info("Validating infrastructure networks.zones[0].name")
		allErrs = append(allErrs, c.validateEnhancedNatGatewayZone(ctx, actor, config.Networks.Zones[0].Name, infra.Spec.Region, field.NewPath("networks", "zones[0]", "name"))...)
	}
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

func (c *configValidator) validateMultiShootVPC(ctx context.Context, actor aliclient.Actor, vpcID string, namespace string, gardenerManagedNATGateway *bool, useCustomRouteTable *bool, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	vswitches, err := actor.FindVSwitchesByVPC(ctx, vpcID)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(fldPath, fmt.Errorf("validateMultiShootVPC FindVSwitchesByVPC %s failed: %+v", vpcID, err)))
		return allErrs
	}

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
