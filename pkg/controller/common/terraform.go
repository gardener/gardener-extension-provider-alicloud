// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package common

import (
	"encoding/json"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/imagevector"

	"github.com/gardener/gardener/extensions/pkg/terraformer"
	"github.com/gardener/gardener/pkg/logger"
	"k8s.io/client-go/rest"
)

const (
	TerraformVarAccessKeyID     = "TF_VAR_ACCESS_KEY_ID"
	TerraformVarAccessKeySecret = "TF_VAR_ACCESS_KEY_SECRET"
	TerraformProvider           = "provider.alicloud"
)

type tfState struct {
	Resources []tfStateResource `json:"resources"`
}

type tfStateResource struct {
	Mode      string        `json:"mode"`
	Type      string        `json:"type"`
	Name      string        `json:"name"`
	Provider  string        `json:"provider"`
	Instances []interface{} `json:"instances"`
}

// NewTerraformer creates a new Terraformer.
func NewTerraformer(factory terraformer.Factory, config *rest.Config, purpose, namespace, name string) (terraformer.Terraformer, error) {
	tf, err := factory.NewForConfig(logger.NewLogger("info"), config, purpose, namespace, name, imagevector.TerraformerImage())
	if err != nil {
		return nil, err
	}

	return tf.
		SetTerminationGracePeriodSeconds(630).
		SetDeadlineCleaning(5 * time.Minute).
		SetDeadlinePod(15 * time.Minute), nil
}

// NewTerraformerWithAuth creates a new Terraformer and initializes it with the credentials.
func NewTerraformerWithAuth(factory terraformer.Factory, config *rest.Config, purpose, namespace, name string, credentials *alicloud.Credentials) (terraformer.Terraformer, error) {
	tf, err := NewTerraformer(factory, config, purpose, namespace, name)
	if err != nil {
		return nil, err
	}

	return tf.SetVariablesEnvironment(TerraformVariablesEnvironmentFromCredentials(credentials)), nil
}

// TerraformVariablesEnvironmentFromCredentials computes the Terraformer variables environment from the
// given ServiceAccount.
func TerraformVariablesEnvironmentFromCredentials(credentials *alicloud.Credentials) map[string]string {
	return map[string]string{
		TerraformVarAccessKeyID:     credentials.AccessKeyID,
		TerraformVarAccessKeySecret: credentials.AccessKeySecret,
	}
}

// IsStateEmpty checks the Terraformer state: 1. is empty or not; 2. contains resources or not
func IsStateEmpty(tf terraformer.Terraformer) (bool, error) {
	stateConfigMap, err := tf.GetState()
	if err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	if len(stateConfigMap) == 0 {
		return true, nil
	}

	var state tfState
	if err := json.Unmarshal(stateConfigMap, &state); err != nil {
		return false, err
	}

	for _, res := range state.Resources {
		if res.Provider == TerraformProvider && len(res.Instances) > 0 {
			return false, nil
		}
	}

	return true, nil
}
