// SPDX-FileCopyrightText: Contributors to the Gardener project
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
	securityv1alpha1constants "github.com/gardener/gardener/pkg/apis/security/v1alpha1/constants"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudhelper "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
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
func (e *ensurer) EnsureCloudProviderSecret(_ context.Context, _ gcontext.GardenContext, newSecret, _ *corev1.Secret) error {
	if newSecret.Labels != nil && newSecret.Labels[securityv1alpha1constants.LabelWorkloadIdentityProvider] == "alicloud" {
		return e.ensureWorkloadIdentitySecret(newSecret)
	}
	return e.ensureStaticCredentialsSecret(newSecret)
}

func (e *ensurer) ensureWorkloadIdentitySecret(secret *corev1.Secret) error {
	if _, ok := secret.Data[securityv1alpha1constants.DataKeyConfig]; !ok {
		return fmt.Errorf("cloudprovider secret is missing a %q data key", securityv1alpha1constants.DataKeyConfig)
	}
	config, err := alicloudhelper.WorkloadIdentityConfigFromBytes(secret.Data[securityv1alpha1constants.DataKeyConfig])
	if err != nil {
		return fmt.Errorf("could not decode %q as WorkloadIdentityConfig: %w", securityv1alpha1constants.DataKeyConfig, err)
	}
	tokenFilePath := alicloud.WorkloadIdentityMountPath + "/token"
	secret.Data[alicloud.OIDCProviderARNKey] = []byte(config.OIDCProviderARN)
	secret.Data[alicloud.RoleARNKey] = []byte(config.RoleARN)
	secret.Data[alicloud.OIDCTokenFileKey] = []byte(tokenFilePath)
	secret.Data[alicloud.CredentialsFile] = []byte(buildOIDCCredentialsFile(config.OIDCProviderARN, config.RoleARN, tokenFilePath))
	return nil
}

func (e *ensurer) ensureStaticCredentialsSecret(secret *corev1.Secret) error {
	if _, ok := secret.Data[alicloud.AccessKeyID]; !ok {
		return fmt.Errorf("could not mutate cloudprovider secret as %q field is missing", alicloud.AccessKeyID)
	}
	if _, ok := secret.Data[alicloud.AccessKeySecret]; !ok {
		return fmt.Errorf("could not mutate cloudprovider secret as %q field is missing", alicloud.AccessKeySecret)
	}
	e.logger.V(5).Info("mutate cloudprovider secret", "namespace", secret.Namespace, "name", secret.Name)
	secret.Data[alicloud.CredentialsFile] = []byte("[default]\n" +
		"type = access_key\n" +
		fmt.Sprintf("access_key_id = %s\n", string(secret.Data[alicloud.AccessKeyID])) +
		fmt.Sprintf("access_key_secret = %s", string(secret.Data[alicloud.AccessKeySecret])),
	)
	return nil
}

// buildOIDCCredentialsFile returns an Alicloud SDK INI credentials file for OIDC role ARN auth.
// The CSI controller (seed-side) reads this file to assume the RAM role via RRSA.
func buildOIDCCredentialsFile(oidcProviderARN, roleARN, tokenFilePath string) string {
	return "[default]\n" +
		"type = oidc_role_arn\n" +
		fmt.Sprintf("oidc_provider_arn = %s\n", oidcProviderARN) +
		fmt.Sprintf("role_arn = %s\n", roleARN) +
		fmt.Sprintf("oidc_token_file = %s", tokenFilePath)
}
