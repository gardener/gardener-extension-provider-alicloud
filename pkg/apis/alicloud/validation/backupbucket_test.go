// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisali "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
)

var _ = Describe("BackupBucket", func() {
	var fldPath *field.Path

	BeforeEach(func() {
		fldPath = field.NewPath("spec")
	})

	Describe("ValidateBackupBucketConfig", func() {
		DescribeTable("validation cases",
			func(config *apisali.BackupBucketConfig, wantErr bool, errMsg string) {
				errs := ValidateBackupBucketConfig(config, fldPath)
				if wantErr {
					Expect(errs).NotTo(BeEmpty())
					Expect(errs[0].Error()).To(ContainSubstring(errMsg))
				} else {
					Expect(errs).To(BeEmpty())
				}
			},
			Entry("valid config",
				&apisali.BackupBucketConfig{
					Immutability: &apisali.ImmutableConfig{
						RetentionType:   "bucket",
						RetentionPeriod: 1,
						Locked:          false,
					},
				}, false, ""),
			Entry("nil config",
				&apisali.BackupBucketConfig{
					Immutability: nil,
				}, false, ""),
			Entry("missing retentionType",
				&apisali.BackupBucketConfig{
					Immutability: &apisali.ImmutableConfig{
						RetentionType:   "",
						RetentionPeriod: 1,
						Locked:          false,
					},
				}, true, "must be 'bucket'"),
			Entry("invalid retentionType",
				&apisali.BackupBucketConfig{
					Immutability: &apisali.ImmutableConfig{
						RetentionType:   "invalid",
						RetentionPeriod: 1,
						Locked:          false,
					},
				}, true, "must be 'bucket'"),
			Entry("invalid retentionPeriod",
				&apisali.BackupBucketConfig{
					Immutability: &apisali.ImmutableConfig{
						RetentionType:   "bucket",
						RetentionPeriod: 0,
						Locked:          false,
					},
				}, true, "can't be less than 1 day"),
			Entry("negative retentionPeriod",
				&apisali.BackupBucketConfig{
					Immutability: &apisali.ImmutableConfig{
						RetentionType:   "bucket",
						RetentionPeriod: -1,
						Locked:          false,
					},
				}, true, "can't be less than 1 day"),
		)
	})

	Describe("ValidateBackupBucketConfigUpdate", func() {
		DescribeTable("Valid update scenarios",
			func(oldConfig, newConfig apisali.BackupBucketConfig) {
				Expect(ValidateBackupBucketConfigUpdate(&oldConfig, &newConfig, fldPath).ToAggregate()).NotTo(HaveOccurred())
			},
			Entry("Immutable settings unchanged",
				generateBackupBucketConfig("bucket", 1, false, true),
				generateBackupBucketConfig("bucket", 1, false, true),
			),
			Entry("Retention period increased while unlocked",
				generateBackupBucketConfig("bucket", 1, false, true),
				generateBackupBucketConfig("bucket", 2, false, true),
			),
			Entry("Retention period increased while locked",
				generateBackupBucketConfig("bucket", 1, true, true),
				generateBackupBucketConfig("bucket", 2, true, true),
			),
			Entry("Adding immutability to an existing bucket",
				generateBackupBucketConfig("", 0, false, false),
				generateBackupBucketConfig("bucket", 1, false, true),
			),
			Entry("Adding immutability with locked set to true",
				generateBackupBucketConfig("", 0, false, false),
				generateBackupBucketConfig("bucket", 1, true, true),
			),
			Entry("Retention period exactly at minimum 1 day",
				generateBackupBucketConfig("bucket", 1, false, true),
				generateBackupBucketConfig("bucket", 1, false, true),
			),
			Entry("Transitioning from locked=false to locked=true",
				generateBackupBucketConfig("bucket", 1, false, true),
				generateBackupBucketConfig("bucket", 1, true, true),
			),
			// To be removed later
			Entry("Disabling immutability when not locked",
				generateBackupBucketConfig("bucket", 1, false, true),
				generateBackupBucketConfig("", 0, false, false),
			),
			Entry("Backup not configured",
				generateBackupBucketConfig("", 0, false, false),
				generateBackupBucketConfig("", 0, false, false),
			),
		)

		DescribeTable("Invalid update scenarios",
			func(oldConfig, newConfig apisali.BackupBucketConfig, expectedError string) {
				err := ValidateBackupBucketConfigUpdate(&oldConfig, &newConfig, fldPath).ToAggregate()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring(expectedError)))
			},
			// TODO: @ishan16696 as currently disabling is allowed
			// Entry("Disabling immutable settings is not allowed if locked",
			// generateBackupBucketConfig("bucket", 2, true, true),
			// generateBackupBucketConfig("", 0, false, false),
			// 	"immutability cannot be disabled once it is locked",
			// ),
			Entry("Unlocking a locked retention policy is not allowed",
				generateBackupBucketConfig("bucket", 1, true, true),
				generateBackupBucketConfig("bucket", 1, false, true),
				"immutable retention policy lock cannot be unlocked once it is locked",
			),
			Entry("Reducing retention period is not allowed when unlocked",
				generateBackupBucketConfig("bucket", 2, false, true),
				generateBackupBucketConfig("bucket", 1, false, true),
				"reducing the retention period from",
			),
			Entry("Reducing retention period is not allowed when locked",
				generateBackupBucketConfig("bucket", 2, true, true),
				generateBackupBucketConfig("bucket", 1, true, true),
				"reducing the retention period from",
			),
		)
	})

	Describe("ValidateBackupBucketCredentialsRef", func() {
		BeforeEach(func() {
			fldPath = field.NewPath("spec", "credentialsRef")
		})

		It("should forbid nil credentialsRef", func() {
			errs := ValidateBackupBucketCredentialsRef(nil, fldPath)
			Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("spec.credentialsRef"),
				"Detail": Equal("must be set"),
			}))))
		})

		It("should forbid v1.ConfigMap credentials", func() {
			credsRef := &corev1.ObjectReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "my-creds",
				Namespace:  "my-namespace",
			}
			errs := ValidateBackupBucketCredentialsRef(credsRef, fldPath)
			Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeNotSupported),
				"Field":  Equal("spec.credentialsRef"),
				"Detail": Equal("supported values: \"/v1, Kind=Secret\""),
			}))))
		})

		It("should forbid security.gardener.cloud/v1alpha1.WorkloadIdentity credentials", func() {
			credsRef := &corev1.ObjectReference{
				APIVersion: "security.gardener.cloud/v1alpha1",
				Kind:       "WorkloadIdentity",
				Name:       "my-creds",
				Namespace:  "my-namespace",
			}
			errs := ValidateBackupBucketCredentialsRef(credsRef, fldPath)
			Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeNotSupported),
				"Field":  Equal("spec.credentialsRef"),
				"Detail": Equal("supported values: \"/v1, Kind=Secret\""),
			}))))
		})

		It("should allow v1.Secret credentials", func() {
			credsRef := &corev1.ObjectReference{
				APIVersion: "v1",
				Kind:       "Secret",
				Name:       "my-creds",
				Namespace:  "my-namespace",
			}
			errs := ValidateBackupBucketCredentialsRef(credsRef, fldPath)
			Expect(errs).To(BeEmpty())
		})
	})
})

// Helper function to generate Seed objects
func generateBackupBucketConfig(retentionType string, retentionPeriod int, locked bool, isImmutableConfigured bool) apisali.BackupBucketConfig {
	if !isImmutableConfigured {
		return apisali.BackupBucketConfig{}
	}

	config := apisali.BackupBucketConfig{Immutability: &apisali.ImmutableConfig{}}

	if retentionType != "" {
		config.Immutability.RetentionType = apisali.RetentionType(retentionType)
	}

	config.Immutability.RetentionPeriod = retentionPeriod
	config.Immutability.Locked = locked

	return config
}
