// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"regexp"

	"k8s.io/apimachinery/pkg/util/validation/field"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
)

var (
	// roleARNRegex validates Alibaba Cloud RAM role ARNs.
	// Role name rules per CreateRole API: 1-64 chars, letters/digits/periods/hyphens only.
	// Ref: https://www.alibabacloud.com/help/en/ram/developer-reference/api-ram-2015-05-01-createrole
	roleARNRegex = regexp.MustCompile(`^acs:ram::\d+:role/[a-zA-Z0-9.-]{1,64}$`)

	// oidcProviderARNRegex validates Alibaba Cloud RAM OIDC provider ARNs.
	// Provider name rules are not explicitly documented; using the same character set as role names.
	oidcProviderARNRegex = regexp.MustCompile(`^acs:ram::\d+:oidc-provider/[a-zA-Z0-9.-]{1,64}$`)
)

// ValidateWorkloadIdentityConfig checks whether the given WorkloadIdentityConfig contains valid values.
func ValidateWorkloadIdentityConfig(config *apisalicloud.WorkloadIdentityConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if config.OIDCProviderARN == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("oidcProviderARN"), "cannot be empty"))
	} else if !oidcProviderARNRegex.MatchString(config.OIDCProviderARN) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("oidcProviderARN"), config.OIDCProviderARN,
			`does not match expected format acs:ram::<accountID>:oidc-provider/<name>`))
	}

	if config.RoleARN == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("roleARN"), "cannot be empty"))
	} else if !roleARNRegex.MatchString(config.RoleARN) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("roleARN"), config.RoleARN,
			`does not match expected format acs:ram::<accountID>:role/<name>`))
	}

	return allErrs
}

// ValidateWorkloadIdentityConfigUpdate validates an update to a WorkloadIdentityConfig.
func ValidateWorkloadIdentityConfigUpdate(_, newConfig *apisalicloud.WorkloadIdentityConfig, fldPath *field.Path) field.ErrorList {
	return ValidateWorkloadIdentityConfig(newConfig, fldPath)
}
