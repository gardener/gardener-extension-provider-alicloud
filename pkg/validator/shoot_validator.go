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

package validator

import (
	"context"
	"reflect"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	alicloudvalidation "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"

	"github.com/gardener/gardener/pkg/apis/core"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (v *Shoot) validateShoot(ctx context.Context, shoot *core.Shoot, infraConfig *alicloud.InfrastructureConfig) error {
	// Network validation
	networkPath := field.NewPath("spec", "networking")

	if errList := alicloudvalidation.ValidateNetworking(shoot.Spec.Networking, networkPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	// Provider validation
	fldPath := field.NewPath("spec", "provider")

	if errList := alicloudvalidation.ValidateInfrastructureConfig(infraConfig, shoot.Spec.Networking.Nodes, shoot.Spec.Networking.Pods, shoot.Spec.Networking.Services); len(errList) != 0 {
		return errList.ToAggregate()
	}

	// ControlPlaneConfig
	if shoot.Spec.Provider.ControlPlaneConfig != nil {
		if _, err := decodeControlPlaneConfig(v.decoder, shoot.Spec.Provider.ControlPlaneConfig, fldPath.Child("controlPlaneConfig")); err != nil {
			return err
		}
	}

	// Shoot workers
	if errList := alicloudvalidation.ValidateWorkers(shoot.Spec.Provider.Workers, infraConfig.Networks.Zones, fldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	return nil
}

func (v *Shoot) validateShootUpdate(ctx context.Context, oldShoot, shoot *core.Shoot) error {
	var (
		fldPath            = field.NewPath("spec", "provider")
		infraConfigFldPath = fldPath.Child("infrastructureConfig")
	)

	// InfrastructureConfig update
	infraConfig, err := checkAndDecodeInfrastructureConfig(v.decoder, shoot.Spec.Provider.InfrastructureConfig, infraConfigFldPath)
	if err != nil {
		return err
	}

	oldInfraConfig, err := checkAndDecodeInfrastructureConfig(v.decoder, oldShoot.Spec.Provider.InfrastructureConfig, infraConfigFldPath)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldInfraConfig, infraConfig) {
		if errList := alicloudvalidation.ValidateInfrastructureConfigUpdate(oldInfraConfig, infraConfig); len(errList) != 0 {
			return errList.ToAggregate()
		}
	}

	if errList := alicloudvalidation.ValidateWorkersUpdate(oldShoot.Spec.Provider.Workers, shoot.Spec.Provider.Workers, fldPath.Child("workers")); len(errList) != 0 {
		return errList.ToAggregate()
	}

	return v.validateShoot(ctx, shoot, infraConfig)
}

func (v *Shoot) validateShootCreation(ctx context.Context, shoot *core.Shoot) error {
	fldPath := field.NewPath("spec", "provider")
	infraConfig, err := checkAndDecodeInfrastructureConfig(v.decoder, shoot.Spec.Provider.InfrastructureConfig, fldPath.Child("infrastructureConfig"))
	if err != nil {
		return err
	}

	return v.validateShoot(ctx, shoot, infraConfig)
}
