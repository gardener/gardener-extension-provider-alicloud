// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
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
