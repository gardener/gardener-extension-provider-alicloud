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
	"fmt"
	"reflect"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	alicloudvalidation "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	fldPath            = field.NewPath("spec")
	networkingFldPath  = fldPath.Child("networking")
	providerFldPath    = fldPath.Child("provider")
	infraConfigFldPath = providerFldPath.Child("infrastructureConfig")
	cpConfigFldPath    = providerFldPath.Child("controlPlaneConfig")
	workersFldPath     = providerFldPath.Child("workers")
)

// NewShootValidator returns a new instance of a shoot validator.
func NewShootValidator() extensionswebhook.Validator {
	return &shoot{}
}

type shoot struct {
	client         client.Client
	apiReader      client.Reader
	decoder        runtime.Decoder
	lenientDecoder runtime.Decoder
}

// InjectScheme injects the given scheme into the validator.
func (s *shoot) InjectScheme(scheme *runtime.Scheme) error {
	s.decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
	s.lenientDecoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
	return nil
}

// InjectClient injects the given client into the validator.
func (s *shoot) InjectClient(client client.Client) error {
	s.client = client
	return nil
}

// InjectAPIReader injects the given apiReader into the validator.
func (s *shoot) InjectAPIReader(apiReader client.Reader) error {
	s.apiReader = apiReader
	return nil
}

// Validate validates the given shoot object.
func (s *shoot) Validate(ctx context.Context, new, old client.Object) error {
	shoot, ok := new.(*core.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	if old != nil {
		oldShoot, ok := old.(*core.Shoot)
		if !ok {
			return fmt.Errorf("wrong object type %T for old object", old)
		}
		return s.validateShootUpdate(ctx, oldShoot, shoot)
	}

	return s.validateShootCreation(ctx, shoot)
}

func (s *shoot) validateShoot(_ context.Context, shoot *core.Shoot, infraConfig *alicloud.InfrastructureConfig, cpConfig *alicloud.ControlPlaneConfig) error {
	// Network validation
	if errList := alicloudvalidation.ValidateNetworking(shoot.Spec.Networking, networkingFldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	// Provider validation
	if errList := alicloudvalidation.ValidateInfrastructureConfig(infraConfig, shoot.Spec.Networking.Nodes, shoot.Spec.Networking.Pods, shoot.Spec.Networking.Services); len(errList) != 0 {
		return errList.ToAggregate()
	}
	if cpConfig != nil {
		if errList := alicloudvalidation.ValidateControlPlaneConfig(cpConfig, shoot.Spec.Kubernetes.Version, cpConfigFldPath); len(errList) != 0 {
			return errList.ToAggregate()
		}
	}

	// Shoot workers
	if errList := alicloudvalidation.ValidateWorkers(shoot.Spec.Provider.Workers, infraConfig.Networks.Zones, workersFldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	return nil
}

func (s *shoot) validateShootUpdate(ctx context.Context, oldShoot, shoot *core.Shoot) error {
	// Decode the new infrastructure config
	infraConfig, err := checkAndDecodeInfrastructureConfig(s.decoder, shoot.Spec.Provider.InfrastructureConfig, infraConfigFldPath)
	if err != nil {
		return err
	}

	// Decode the old infrastructure config
	oldInfraConfig, err := checkAndDecodeInfrastructureConfig(s.lenientDecoder, oldShoot.Spec.Provider.InfrastructureConfig, infraConfigFldPath)
	if err != nil {
		return err
	}

	// Decode the new controlplane config
	var cpConfig *alicloud.ControlPlaneConfig
	if shoot.Spec.Provider.ControlPlaneConfig != nil {
		// We use "lenientDecoder" because the "zone" field of "ControlPlaneConfig" was removed with https://github.com/gardener/gardener-extension-provider-alicloud/pull/64
		// but still Shoots in Gardener environments may contain the legacy "zone" field.
		// We cannot use strict "decoder" because it will complain that the "zone" field is specified but it is actually an invalid.
		// Let's use "lenientDecoder" for now to make the migration smoother for such Shoots.
		// TODO: consider enabling the strict "decoder" in a future release.
		cpConfig, err = decodeControlPlaneConfig(s.lenientDecoder, shoot.Spec.Provider.ControlPlaneConfig, cpConfigFldPath)
		if err != nil {
			return err
		}
	}

	if !reflect.DeepEqual(oldInfraConfig, infraConfig) {
		if errList := alicloudvalidation.ValidateInfrastructureConfigUpdate(oldInfraConfig, infraConfig); len(errList) != 0 {
			return errList.ToAggregate()
		}
	}

	if errList := alicloudvalidation.ValidateWorkersUpdate(oldShoot.Spec.Provider.Workers, shoot.Spec.Provider.Workers, providerFldPath.Child("workers")); len(errList) != 0 {
		return errList.ToAggregate()
	}

	if errList := alicloudvalidation.ValidateNetworkingUpdate(oldShoot.Spec.Networking, shoot.Spec.Networking, networkingFldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	return s.validateShoot(ctx, shoot, infraConfig, cpConfig)
}

func (s *shoot) validateShootCreation(ctx context.Context, shoot *core.Shoot) error {
	// Decode the infrastructure config
	infraConfig, err := checkAndDecodeInfrastructureConfig(s.decoder, shoot.Spec.Provider.InfrastructureConfig, infraConfigFldPath)
	if err != nil {
		return err
	}

	// Decode the controlplane config
	var cpConfig *alicloud.ControlPlaneConfig
	if shoot.Spec.Provider.ControlPlaneConfig != nil {
		// TODO: consider enabling the strict "decoder" in a future release (see above).
		cpConfig, err = decodeControlPlaneConfig(s.lenientDecoder, shoot.Spec.Provider.ControlPlaneConfig, cpConfigFldPath)
		if err != nil {
			return err
		}
	}

	if err := s.validateShoot(ctx, shoot, infraConfig, cpConfig); err != nil {
		return err
	}

	return s.validateShootSecret(ctx, shoot)
}

func (s *shoot) validateShootSecret(ctx context.Context, shoot *core.Shoot) error {
	var (
		secretBinding    = &gardencorev1beta1.SecretBinding{}
		secretBindingKey = kutil.Key(shoot.Namespace, shoot.Spec.SecretBindingName)
	)
	if err := kutil.LookupObject(ctx, s.client, s.apiReader, secretBindingKey, secretBinding); err != nil {
		return err
	}

	var (
		secret    = &corev1.Secret{}
		secretKey = kutil.Key(secretBinding.SecretRef.Namespace, secretBinding.SecretRef.Name)
	)
	// Explicitly use the client.Reader to prevent controller-runtime to start Informer for Secrets
	// under the hood. The latter increases the memory usage of the component.
	if err := s.apiReader.Get(ctx, secretKey, secret); err != nil {
		return err
	}

	return alicloudvalidation.ValidateCloudProviderSecret(secret)
}
