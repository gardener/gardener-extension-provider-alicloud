// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator_test

import (
	"context"
	"encoding/json"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	core "github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	mockclient "github.com/gardener/gardener/third_party/mock/controller-runtime/client"
	mockmanager "github.com/gardener/gardener/third_party/mock/controller-runtime/manager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/admission/validator"

	apisali "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	apisaliv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
)

var _ = Describe("Seed Validator", func() {
	var (
		ctrl          *gomock.Controller
		mgr           *mockmanager.MockManager
		c             *mockclient.MockClient
		seedValidator extensionswebhook.Validator
		scheme        *runtime.Scheme
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		scheme = runtime.NewScheme()
		Expect(core.AddToScheme(scheme)).To(Succeed())
		Expect(apisali.AddToScheme(scheme)).To(Succeed())
		Expect(apisaliv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(gardencorev1beta1.AddToScheme(scheme)).To(Succeed())
		c = mockclient.NewMockClient(ctrl)

		mgr = mockmanager.NewMockManager(ctrl)
		mgr.EXPECT().GetScheme().Return(scheme).AnyTimes()
		mgr.EXPECT().GetClient().Return(c).AnyTimes()
		seedValidator = validator.NewSeedValidator(mgr)
	})

	// Helper function to generate Seed objects
	generateSeed := func(retentionType string, retentionPeriod int, locked bool, isImmutableConfigured bool) *core.Seed {
		var config *runtime.RawExtension
		if isImmutableConfigured {

			immutability := make(map[string]interface{})
			if retentionType != "" {
				immutability["retentionType"] = retentionType
			}
			immutability["retentionPeriod"] = retentionPeriod
			immutability["locked"] = locked

			backupBucketConfig := map[string]interface{}{
				"apiVersion":   "alicloud.provider.extensions.gardener.cloud/v1alpha1",
				"kind":         "BackupBucketConfig",
				"immutability": immutability,
			}
			raw, err := json.Marshal(backupBucketConfig)
			Expect(err).NotTo(HaveOccurred())
			config = &runtime.RawExtension{
				Raw: raw,
			}
		} else {
			config = nil
		}

		var backup *core.Backup
		if config != nil {
			backup = &core.Backup{
				ProviderConfig: config,
			}
		}

		return &core.Seed{
			Spec: core.SeedSpec{
				Backup: backup,
			},
		}
	}

	Describe("ValidateUpdate", func() {
		DescribeTable("Valid update scenarios",
			func(oldSeed, newSeed *core.Seed) {
				err := seedValidator.Validate(context.Background(), newSeed, oldSeed)
				Expect(err).NotTo(HaveOccurred())
			},
			Entry("Immutable settings unchanged",
				generateSeed("bucket", 1, false, true),
				generateSeed("bucket", 1, false, true),
			),
			Entry("Retention period increased while unlocked",
				generateSeed("bucket", 1, false, true),
				generateSeed("bucket", 2, false, true),
			),
			Entry("Retention period increased while locked",
				generateSeed("bucket", 1, true, true),
				generateSeed("bucket", 2, true, true),
			),
			Entry("Adding immutability to an existing bucket",
				generateSeed("", 0, false, false),
				generateSeed("bucket", 1, false, true),
			),
			Entry("Adding immutability with locked set to true",
				generateSeed("", 0, false, false),
				generateSeed("bucket", 1, true, true),
			),
			Entry("Retention period exactly at minimum 1 day",
				generateSeed("bucket", 1, false, true),
				generateSeed("bucket", 1, false, true),
			),
			Entry("Transitioning from locked=false to locked=true",
				generateSeed("bucket", 1, false, true),
				generateSeed("bucket", 1, true, true),
			),
			// To be removed later
			Entry("Disabling immutability when not locked",
				generateSeed("bucket", 1, false, true),
				generateSeed("", 0, false, false),
			),
			Entry("Backup not configured",
				generateSeed("", 0, false, false),
				generateSeed("", 0, false, false),
			),
		)

		DescribeTable("Invalid update scenarios",
			func(oldSeed, newSeed *core.Seed, expectedError string) {
				err := seedValidator.Validate(context.Background(), newSeed, oldSeed)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectedError))
			},
			// TODO: @ishan16696 as currently disabling is allowed
			// Entry("Disabling immutable settings is not allowed if locked",
			// generateSeed("bucket", 2, true, true),
			// generateSeed("", 0, false, false),
			// 	"immutability cannot be disabled once it is locked",
			// ),
			Entry("Unlocking a locked retention policy is not allowed",
				generateSeed("bucket", 1, true, true),
				generateSeed("bucket", 1, false, true),
				"immutable retention policy lock cannot be unlocked once it is locked",
			),
			Entry("Reducing retention period is not allowed when unlocked",
				generateSeed("bucket", 2, false, true),
				generateSeed("bucket", 1, false, true),
				"reducing the retention period from",
			),
			Entry("Reducing retention period is not allowed when locked",
				generateSeed("bucket", 2, true, true),
				generateSeed("bucket", 1, true, true),
				"reducing the retention period from",
			),
			Entry("Changing retentionType is not allowed",
				generateSeed("bucket", 1, true, true),
				generateSeed("object", 1, true, true),
				"must be 'bucket'",
			),
			Entry("Retention period below minimum is not allowed",
				generateSeed("bucket", 1, false, true),
				generateSeed("bucket", 0, false, true),
				"it can't be less than 1 day",
			),
		)
	})

	Describe("ValidateCreate", func() {
		DescribeTable("Valid creation scenarios",
			func(newSeed *core.Seed) {
				err := seedValidator.Validate(context.Background(), newSeed, nil)
				Expect(err).NotTo(HaveOccurred())
			},
			Entry("Creation with valid immutable settings",
				generateSeed("bucket", 2, false, true),
			),
			Entry("Creation without immutable settings",
				generateSeed("", 0, false, false),
			),
			Entry("Creation with locked immutable settings",
				generateSeed("bucket", 1, true, true),
			),
			Entry("Retention period exactly at minimum 1 day",
				generateSeed("bucket", 1, false, true),
			),
		)

		DescribeTable("Invalid creation scenarios",
			func(newSeed *core.Seed, expectedError string) {
				err := seedValidator.Validate(context.Background(), newSeed, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectedError))
			},
			Entry("Invalid retention type",
				generateSeed("invalid", 1, false, true),
				"must be 'bucket'",
			),
			Entry("Invalid retention period",
				&core.Seed{
					Spec: core.SeedSpec{
						Backup: &core.Backup{
							ProviderConfig: &runtime.RawExtension{
								Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": -1, "locked": false }}`),
							},
						},
					},
				},
				"can't be less than 1 day",
			),
			Entry("Retention period below minimum when not locked",
				generateSeed("bucket", 0, false, true),
				"can't be less than 1 day",
			),
			Entry("Retention period below minimum when locked",
				generateSeed("bucket", 0, true, true),
				"can't be less than 1 day",
			),
		)
	})
})
