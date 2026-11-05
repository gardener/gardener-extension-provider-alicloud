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

// ValidateCSIForWorkloadIdentity validates that csi.enableLabeler is true when WorkloadIdentity is used.
func ValidateCSIForWorkloadIdentity(controlPlaneConfig *apisalicloud.ControlPlaneConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	labelerEnabled := controlPlaneConfig != nil &&
		controlPlaneConfig.CSI != nil &&
		controlPlaneConfig.CSI.EnableLabeler != nil &&
		*controlPlaneConfig.CSI.EnableLabeler
	if !labelerEnabled {
		allErrs = append(allErrs, field.Required(fldPath.Child("csi", "enableLabeler"),
			"must be true when WorkloadIdentity is used for the shoot"))
	}
	return allErrs
}
