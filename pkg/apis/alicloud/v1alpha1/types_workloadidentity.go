// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkloadIdentityConfig contains the configuration for a workload identity.
type WorkloadIdentityConfig struct {
	metav1.TypeMeta `json:",inline"`

	// OIDCProviderARN is the ARN of the OIDC identity provider registered in Alicloud RAM.
	// Format: acs:ram::<accountID>:oidc-provider/<name>
	OIDCProviderARN string `json:"oidcProviderARN"`

	// RoleARN is the ARN of the RAM role to be assumed via STS AssumeRoleWithOIDC.
	// Format: acs:ram::<accountID>:role/<name>
	RoleARN string `json:"roleARN"`
}
