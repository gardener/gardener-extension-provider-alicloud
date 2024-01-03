// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	"context"
	"fmt"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudapi "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	"github.com/gardener/gardener/extensions/pkg/util"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// configValidator implements ConfigValidator for alicloud infrastructure resources.
type configValidator struct {
	mgr    manager.Manager
	logger logr.Logger
}

// NewConfigValidator creates a new ConfigValidator.
func NewConfigValidator(mgr manager.Manager, logger logr.Logger) infrastructure.ConfigValidator {
	return &configValidator{
		mgr:    mgr,
		logger: logger.WithName("alicloud-infrastructure-config-validator"),
	}
}

// Validate validates the provider config of the given infrastructure resource with the cloud provider.
func (c *configValidator) Validate(ctx context.Context, infra *extensionsv1alpha1.Infrastructure) field.ErrorList {
	allErrs := field.ErrorList{}

	logger := c.logger.WithValues("infrastructure", client.ObjectKeyFromObject(infra))

	config := &alicloudapi.InfrastructureConfig{}
	decoder := serializer.NewCodecFactory(c.mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder()
	if err := util.Decode(decoder, infra.Spec.ProviderConfig.Raw, config); err != nil {
		allErrs = append(allErrs, field.InternalError(nil, err))
		return allErrs
	}

	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, c.mgr.GetClient(), &infra.Spec.SecretRef)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, fmt.Errorf("could not get Alicloud credentials: %+v", err)))
		return allErrs
	}
	actor, err := aliclient.NewActor(credentials.AccessKeyID, credentials.AccessKeySecret, infra.Spec.Region)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, fmt.Errorf("create aliclient actor failed: %+v", err)))
		return allErrs
	}

	// Validate infrastructure config
	createManagedNATGateway := true
	if config.Networks.VPC.ID != nil {
		logger.Info("Validating infrastructure networks.vpc.id")
		allErrs = append(allErrs, c.validateVPC(ctx, actor, *config.Networks.VPC.ID, infra.Spec.Region, field.NewPath("networks", "vpc", "id"))...)
		if config.Networks.VPC.GardenerManagedNATGateway == nil || !*config.Networks.VPC.GardenerManagedNATGateway {
			createManagedNATGateway = false
			logger.Info("Validating infrastructure networks.vpc.gardenerManagedNATGateway")
			allErrs = append(allErrs, c.validateUserNatGateway(ctx, actor, *config.Networks.VPC.ID, infra.Spec.Region, field.NewPath("networks", "vpc", "gardenerManagedNATGateway"))...)
		}
	}
	if createManagedNATGateway {
		logger.Info("Validating infrastructure networks.zones[0].name")
		allErrs = append(allErrs, c.validateEnhancedNatGatewayZone(ctx, actor, config.Networks.Zones[0].Name, infra.Spec.Region, field.NewPath("networks", "zones[0]", "name"))...)
	}

	for _, zone := range config.Networks.Zones {
		if zone.NatGateway != nil && zone.NatGateway.EIPAllocationID != nil {
			logger.Info("Validating infrastructure networks.zones[].natGatewayid.eipAllocationID")
			allErrs = append(allErrs, c.validateEIP(ctx, actor, *zone.NatGateway.EIPAllocationID, field.NewPath("networks", "zones[]", "natGateway", "eipAllocationID"))...)
		}
	}

	return allErrs
}

func (c *configValidator) validateVPC(ctx context.Context, actor aliclient.Actor, vpcID, region string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// check vpc if exists
	vpc, err := actor.GetVpc(ctx, vpcID)
	if err != nil || vpc == nil {
		allErrs = append(allErrs, field.NotFound(fldPath, vpcID))
	}
	return allErrs
}

func (c *configValidator) validateUserNatGateway(ctx context.Context, actor aliclient.Actor, vpcID, region string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	gw, err := actor.FindNatGatewayByVPC(ctx, vpcID)
	if err != nil || gw == nil {
		allErrs = append(allErrs, field.Invalid(fldPath, vpcID, "no user natgateway found"))
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
		allErrs = append(allErrs, field.NotSupported(fldPath, zone, validZones))
	}
	return allErrs
}
