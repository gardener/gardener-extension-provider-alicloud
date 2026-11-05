// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	featurevalidation "github.com/gardener/gardener/pkg/utils/validation/features"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
)

// ValidateControlPlaneConfig validates a ControlPlaneConfig object.
func ValidateControlPlaneConfig(controlPlaneConfig *apisalicloud.ControlPlaneConfig, version string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if controlPlaneConfig.CloudControllerManager != nil {
		allErrs = append(allErrs, featurevalidation.ValidateFeatureGates(controlPlaneConfig.CloudControllerManager.FeatureGates, version, fldPath.Child("cloudControllerManager", "featureGates"))...)
	}

	return allErrs
}

// ValidateCSIForWorkloadIdentity validates that the CSI configuration is compatible with WorkloadIdentity.
// When WI is used, useLabeler = enableADController && enableLabeler must be true so that the node
// plugin runs without cloud API access. enableADController defaults to true (nil is acceptable);
// enableLabeler must be explicitly set to true.
func ValidateCSIForWorkloadIdentity(controlPlaneConfig *apisalicloud.ControlPlaneConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	csiPath := fldPath.Child("csi")

	// enableADController defaults to true when nil; only invalid if explicitly set to false.
	if controlPlaneConfig != nil &&
		controlPlaneConfig.CSI != nil &&
		controlPlaneConfig.CSI.EnableADController != nil &&
		!*controlPlaneConfig.CSI.EnableADController {
		allErrs = append(allErrs, field.Invalid(csiPath.Child("enableADController"), false,
			"must not be false when WorkloadIdentity is used for the shoot"))
	}

	labelerEnabled := controlPlaneConfig != nil &&
		controlPlaneConfig.CSI != nil &&
		controlPlaneConfig.CSI.EnableLabeler != nil &&
		*controlPlaneConfig.CSI.EnableLabeler
	if !labelerEnabled {
		allErrs = append(allErrs, field.Required(csiPath.Child("enableLabeler"),
			"must be true when WorkloadIdentity is used for the shoot"))
	}

	return allErrs
}
