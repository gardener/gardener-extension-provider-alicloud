// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
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
	if err != nil || vpc == nil {
		allErrs = append(allErrs, field.NotFound(fldPath, vpcID))
		return allErrs
	}
	if checkNatgatewayExists {
		gw, err := actor.FindNatGatewayByVPC(ctx, vpcID)
		if err != nil || gw == nil {
			allErrs = append(allErrs, field.Invalid(fldPath, vpcID, "no user natgateway found"))
		}
	}

	return allErrs
}

func (c *configValidator) validateEIP(ctx context.Context, actor aliclient.Actor, eipId string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	eip, err := actor.GetEIP(ctx, eipId)
	if err != nil || eip == nil {
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
