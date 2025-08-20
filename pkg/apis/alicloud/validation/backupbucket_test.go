// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisali "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
)

var _ = Describe("ValidateBackupBucketConfig", func() {
	var fldPath *field.Path

	BeforeEach(func() {
		fldPath = field.NewPath("spec")
	})

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
