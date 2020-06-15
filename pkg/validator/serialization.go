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
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener/extensions/pkg/util"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
