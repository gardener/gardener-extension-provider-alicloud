// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"github.com/gardener/gardener/extensions/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
)

func decodeControlPlaneConfig(decoder runtime.Decoder, cp *runtime.RawExtension, fldPath *field.Path) (*alicloud.ControlPlaneConfig, error) {
	controlPlaneConfig := &alicloud.ControlPlaneConfig{}
	if err := util.Decode(decoder, cp.Raw, controlPlaneConfig); err != nil {
		return nil, field.Invalid(fldPath, string(cp.Raw), "isn't a supported version")
	}

	return controlPlaneConfig, nil
}

func decodeInfrastructureConfig(decoder runtime.Decoder, infra *runtime.RawExtension, fldPath *field.Path) (*alicloud.InfrastructureConfig, error) {
	infraConfig := &alicloud.InfrastructureConfig{}
	if err := util.Decode(decoder, infra.Raw, infraConfig); err != nil {
		return nil, field.Invalid(fldPath, string(infra.Raw), "isn't a supported version")
	}

	return infraConfig, nil
}

func checkAndDecodeInfrastructureConfig(decoder runtime.Decoder, config *runtime.RawExtension, fldPath *field.Path) (*alicloud.InfrastructureConfig, error) {
	if config == nil {
		return nil, field.Required(fldPath, "InfrastructureConfig must be set for Alicloud shoots")
	}

	infraConfig, err := decodeInfrastructureConfig(decoder, config, fldPath)
	if err != nil {
		return nil, field.Forbidden(fldPath, "not allowed to configure an unsupported infrastructureConfig")
	}
	return infraConfig, nil
}

func decodeCloudProfileConfig(decoder runtime.Decoder, config *runtime.RawExtension) (*alicloud.CloudProfileConfig, error) {
	cloudProfileConfig := &alicloud.CloudProfileConfig{}
	if err := util.Decode(decoder, config.Raw, cloudProfileConfig); err != nil {
		return nil, err
	}
	return cloudProfileConfig, nil
}
