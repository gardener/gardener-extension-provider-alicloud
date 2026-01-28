// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	alicloudvalidation "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"
)

// NewCloudProfileValidator returns a new instance of a cloud profile validator.
func NewCloudProfileValidator(mgr manager.Manager) extensionswebhook.Validator {
	return &cloudProfile{
		decoder: serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
	}
}

type cloudProfile struct {
	decoder runtime.Decoder
}

// Validate validates the given cloud profile objects.
func (cp *cloudProfile) Validate(_ context.Context, newObject, _ client.Object) error {
	cloudProfile, ok := newObject.(*core.CloudProfile)
	if !ok {
		return fmt.Errorf("wrong object type %T", newObject)
	}

	providerConfigPath := field.NewPath("spec").Child("providerConfig")
	if cloudProfile.Spec.ProviderConfig == nil {
		return field.Required(providerConfigPath, "providerConfig must be set for Alicloud cloud profiles")
	}

	cpConfig, err := decodeCloudProfileConfig(cp.decoder, cloudProfile.Spec.ProviderConfig)
	if err != nil {
		return err
	}
	capabilityDefinitions, err := gardencorev1beta1helper.ConvertV1beta1CapabilityDefinitions(cloudProfile.Spec.MachineCapabilities)
	if err != nil {
		return field.InternalError(field.NewPath("spec").Child("machineCapabilities"), err)
	}

	return alicloudvalidation.ValidateCloudProfileConfig(cpConfig, cloudProfile.Spec.MachineImages, capabilityDefinitions, providerConfigPath).ToAggregate()
}
