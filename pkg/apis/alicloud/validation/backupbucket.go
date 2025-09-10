// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisali "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
)

// ValidateBackupBucketConfig validates a BackupBucketConfig object.
func ValidateBackupBucketConfig(backupBucketConfig *apisali.BackupBucketConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if backupBucketConfig == nil || backupBucketConfig.Immutability == nil {
		return allErrs
	}

	// Currently, only RetentionType: "bucket" is supported.
	if backupBucketConfig.Immutability.RetentionType != apisali.BucketLevelImmutability {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("immutability", "retentionType"), backupBucketConfig.Immutability.RetentionType, "must be 'bucket'"))
	}

	// Alicloud OSS immutability period can only be set in days and can't be less than 1 day and must be a positive integer.
	if backupBucketConfig.Immutability.RetentionPeriod < 1 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("immutability", "retentionPeriod"), backupBucketConfig.Immutability.RetentionPeriod, "can only be set in days, hence it can't be less than 1 day"))
	}

	return allErrs
}

// ValidateBackupBucketConfigUpdate validates updates to the BackupBucketConfig.
func ValidateBackupBucketConfigUpdate(oldConfig, newConfig *apisali.BackupBucketConfig, fldPath *field.Path) field.ErrorList {
	var (
		allErrs          = field.ErrorList{}
		immutabilityPath = fldPath.Child("immutability")
	)

	// Note: Right now, immutability can be disabled.
	// TODO: @ishan16696 to remove these conditions "newConfig == nil || newConfig.Immutability == nil" to not allow disablement of immutability settings.
	if oldConfig == nil || oldConfig.Immutability == nil || newConfig == nil || newConfig.Immutability == nil {
		return allErrs
	}

	// TODO: @ishan16696 uncomment this piece of code, so once disablement of the immutability settings on bucket is not allowed.
	/*
		if newConfig == nil || newConfig.Immutability == nil || *newConfig.Immutability == (apisaws.ImmutableConfig{}) {
			allErrs = append(allErrs, field.Invalid(immutabilityPath, newConfig, "immutability cannot be disabled"))
			return allErrs
		}
	*/

	if oldConfig.Immutability.Locked && !newConfig.Immutability.Locked {
		allErrs = append(allErrs, field.Forbidden(immutabilityPath.Child("locked"), "immutable retention policy lock cannot be unlocked once it is locked"))
	}

	if newConfig.Immutability.RetentionPeriod < oldConfig.Immutability.RetentionPeriod {
		allErrs = append(allErrs, field.Forbidden(
			immutabilityPath.Child("retentionPeriod"),
			fmt.Sprintf("reducing the retention period from %v to %v days is prohibited",
				oldConfig.Immutability.RetentionPeriod,
				newConfig.Immutability.RetentionPeriod,
			),
		))
	}

	return allErrs
}

// ValidateBackupBucketCredentialsRef validates credentialsRef is set to supported kind of credentials.
func ValidateBackupBucketCredentialsRef(credentialsRef *corev1.ObjectReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if credentialsRef == nil {
		return append(allErrs, field.Required(fldPath, "must be set"))
	}

	var (
		secretGVK = corev1.SchemeGroupVersion.WithKind("Secret")

		allowedGVKs = sets.New(secretGVK)
		validGVKs   = []string{secretGVK.String()}
	)

	if !allowedGVKs.Has(credentialsRef.GroupVersionKind()) {
		allErrs = append(allErrs, field.NotSupported(fldPath, credentialsRef.String(), validGVKs))
	}

	return allErrs
}
