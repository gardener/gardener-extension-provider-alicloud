// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"context"
	"fmt"
	"reflect"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	gardencorehelper "github.com/gardener/gardener/pkg/apis/core/helper"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	alicloudvalidation "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"
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
func NewShootValidator(mgr manager.Manager) extensionswebhook.Validator {
	alicloudclientFactory := alicloudclient.NewClientFactory()
	return &shoot{
		client:                mgr.GetClient(),
		apiReader:             mgr.GetAPIReader(),
		decoder:               serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
		lenientDecoder:        serializer.NewCodecFactory(mgr.GetScheme()).UniversalDecoder(),
		alicloudClientFactory: alicloudclientFactory,
	}
}

type shoot struct {
	client                client.Client
	apiReader             client.Reader
	decoder               runtime.Decoder
	lenientDecoder        runtime.Decoder
	alicloudClientFactory alicloudclient.ClientFactory
}

// Validate validates the given shoot object.
func (s *shoot) Validate(ctx context.Context, newObject, oldObject client.Object) error {
	shoot, ok := newObject.(*core.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", newObject)
	}

	// skip validation if it's a workerless Shoot
	if gardencorehelper.IsWorkerless(shoot) {
		return nil
	}

	if oldObject != nil {
		oldShoot, ok := oldObject.(*core.Shoot)
		if !ok {
			return fmt.Errorf("wrong object type %T for old object", oldObject)
		}
		return s.validateShootUpdate(ctx, oldShoot, shoot)
	}

	return s.validateShootCreation(ctx, shoot)
}

func (s *shoot) validateShoot(_ context.Context, shoot *core.Shoot, infraConfig *alicloud.InfrastructureConfig, cpConfig *alicloud.ControlPlaneConfig) error {

	if infraConfig != nil {
		// Provider validation
		if errList := alicloudvalidation.ValidateInfrastructureConfig(infraConfig, shoot.Spec.Networking); len(errList) != 0 {
			return errList.ToAggregate()
		}
		// Shoot workers
		if errList := alicloudvalidation.ValidateWorkers(shoot.Spec.Provider.Workers, infraConfig.Networks.Zones, workersFldPath); len(errList) != 0 {
			return errList.ToAggregate()
		}
	}
	if cpConfig != nil {
		if errList := alicloudvalidation.ValidateControlPlaneConfig(cpConfig, shoot.Spec.Kubernetes.Version, cpConfigFldPath); len(errList) != 0 {
			return errList.ToAggregate()
		}
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

	if errList := alicloudvalidation.ValidateNetworking(shoot.Spec.Networking, networkingFldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	return nil
}
