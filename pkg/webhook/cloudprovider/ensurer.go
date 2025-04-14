// Copyright (c) 2025 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package cloudprovider

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/webhook/cloudprovider"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
)

// NewEnsurer creates cloudprovider ensurer.
func NewEnsurer(logger logr.Logger) cloudprovider.Ensurer {
	return &ensurer{
		logger: logger,
	}
}

type ensurer struct {
	logger logr.Logger
}

// EnsureCloudProviderSecret ensures that cloudprovider secret contains
// the shared credentials file.
func (e *ensurer) EnsureCloudProviderSecret(_ context.Context, _ gcontext.GardenContext, new, _ *corev1.Secret) error {
	if _, ok := new.Data[alicloud.AccessKeyID]; !ok {
		return fmt.Errorf("could not mutate cloudprovider secret as %q field is missing", alicloud.AccessKeyID)
	}
	if _, ok := new.Data[alicloud.AccessKeySecret]; !ok {
		return fmt.Errorf("could not mutate cloudprovider secret as %q field is missing", alicloud.AccessKeySecret)
	}

	e.logger.V(5).Info("mutate cloudprovider secret", "namespace", new.Namespace, "name", new.Name)
	new.Data[alicloud.CredentialsFile] = []byte("[default]\n" +
		"type = access_key\n" +
		fmt.Sprintf("access_key_id = %s\n", string(new.Data[alicloud.AccessKeyID])) +
		fmt.Sprintf("access_key_secret = %s", string(new.Data[alicloud.AccessKeySecret])),
	)

	return nil
}
