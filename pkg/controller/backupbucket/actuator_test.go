// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package backupbucket_test

import (
	"context"
	"fmt"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gardener/gardener/extensions/pkg/controller/backupbucket"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	mockclient "github.com/gardener/gardener/third_party/mock/controller-runtime/client"
	mockmanager "github.com/gardener/gardener/third_party/mock/controller-runtime/manager"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	apisali "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	apisalicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/backupbucket"
	mockalicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/mock/provider-alicloud/alicloud/client"
)

const (
	bucketName      = "test-bucket"
	region          = "test-1"
	accessKeyID     = "accessKeyID"
	secretAccessKey = "secretAccessKey"
	name            = "alicloud-operator"
	namespace       = "shoot--test--alicloud"
)

var _ = Describe("Actuator", func() {
	var (
		ctrl                  *gomock.Controller
		c                     *mockclient.MockClient
		mgr                   *mockmanager.MockManager
		sw                    *mockclient.MockStatusWriter
		a                     backupbucket.Actuator
		alicloudClientFactory *mockalicloudclient.MockClientFactory
		ossClient             *mockalicloudclient.MockOSS
		ctx                   context.Context
		logger                logr.Logger
		secret                *corev1.Secret
		secretRef             = corev1.SecretReference{Name: name, Namespace: namespace}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		scheme := runtime.NewScheme()

		Expect(extensionsv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(apisalicloudv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(apisali.AddToScheme(scheme)).To(Succeed())

		c = mockclient.NewMockClient(ctrl)
		mgr = mockmanager.NewMockManager(ctrl)
		mgr.EXPECT().GetClient().Return(c).AnyTimes()
		c.EXPECT().Scheme().Return(scheme).MaxTimes(1)

		sw = mockclient.NewMockStatusWriter(ctrl)
		alicloudClientFactory = mockalicloudclient.NewMockClientFactory(ctrl)
		ossClient = mockalicloudclient.NewMockOSS(ctrl)

		c.EXPECT().Status().Return(sw).AnyTimes()
		sw.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		ctx = context.Background()
		logger = log.Log.WithName("test")

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				alicloud.AccessKeyID:     []byte(accessKeyID),
				alicloud.AccessKeySecret: []byte(secretAccessKey),
			},
		}

		a = NewActuator(mgr, alicloudClientFactory)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#Reconcile", func() {
		var backupBucket *extensionsv1alpha1.BackupBucket

		BeforeEach(func() {
			backupBucket = &extensionsv1alpha1.BackupBucket{
				ObjectMeta: metav1.ObjectMeta{
					Name:      bucketName,
					Namespace: namespace,
				},
				Spec: extensionsv1alpha1.BackupBucketSpec{
					SecretRef: secretRef,
					Region:    region,
				},
			}

			c.EXPECT().Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
				func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
					*obj = *secret
					return nil
				},
			)
		})

		Context("when creation of alicloud's oss client fails", func() {
			It("should return an error", func() {
				alicloudClientFactory.EXPECT().NewOSSClient(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("failed to create alicloud oss client"))

				err := a.Reconcile(ctx, logger, backupBucket)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when decoder failed to decode backupBucket provider config", func() {
			It("should return error", func() {
				// wrong "providerConfig" is passed
				backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
					Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "someField": "someValue"}`),
				}
				alicloudClientFactory.EXPECT().NewOSSClient(gomock.Any(), gomock.Any(), gomock.Any()).Return(ossClient, nil)

				err := a.Reconcile(ctx, logger, backupBucket)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when bucket does not exist", func() {
			BeforeEach(func() {
				alicloudClientFactory.EXPECT().NewOSSClient(gomock.Any(), gomock.Any(), gomock.Any()).Return(ossClient, nil).AnyTimes()
				ossClient.EXPECT().GetBucketInfo(gomock.Any()).DoAndReturn(
					func(_ string, _ ...oss.Option) (*oss.BucketInfo, error) {
						return nil, oss.ServiceError{
							Code: "NoSuchBucket",
						}
					},
				)
				ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
					func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
						return nil, oss.ServiceError{
							Code: "NoSuchWORMConfiguration",
						}
					},
				).AnyTimes()
			})

			It("should return error if creation of bucket fails", func() {
				ossClient.EXPECT().CreateBucketIfNotExists(ctx, gomock.Any()).Return(fmt.Errorf("unable to create bucket"))

				err := a.Reconcile(ctx, logger, backupBucket)
				Expect(err).Should(HaveOccurred())
			})

			It("should create the bucket successfully without bucket lock enabled", func() {
				ossClient.EXPECT().CreateBucketIfNotExists(ctx, gomock.Any()).Return(nil)

				err := a.Reconcile(ctx, logger, backupBucket)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should create the bucket with bucket lock enabled and unlocked retention policy", func() {
				backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
					Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 1, "locked": false }}`),
				}
				ossClient.EXPECT().CreateBucketIfNotExists(ctx, gomock.Any()).Return(nil)
				ossClient.EXPECT().CreateRetentionPolicy(gomock.Any(), gomock.Any()).Return("dummyWormID", nil)

				err := a.Reconcile(ctx, logger, backupBucket)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should create the bucket with bucket lock enabled and locked the retention policy", func() {
				backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
					Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 1, "locked": true }}`),
				}
				ossClient.EXPECT().CreateBucketIfNotExists(ctx, gomock.Any()).Return(nil)
				ossClient.EXPECT().CreateRetentionPolicy(gomock.Any(), gomock.Any()).Return("dummyWormID", nil)
				ossClient.EXPECT().LockRetentionPolicy(gomock.Any(), gomock.Any()).Return(nil)

				err := a.Reconcile(ctx, logger, backupBucket)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should return error if creation of bucket succeeds but unable to add retention policy", func() {
				backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
					Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 1, "locked": false }}`),
				}
				ossClient.EXPECT().CreateBucketIfNotExists(ctx, gomock.Any()).Return(nil)
				ossClient.EXPECT().CreateRetentionPolicy(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ string, _ int, _ ...oss.Option) (string, error) {
						return "", fmt.Errorf("unable to create retention policy on bucket")
					},
				)

				err := a.Reconcile(ctx, logger, backupBucket)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when bucket exist", func() {
			BeforeEach(func() {
				alicloudClientFactory.EXPECT().NewOSSClient(gomock.Any(), gomock.Any(), gomock.Any()).Return(ossClient, nil).AnyTimes()
				ossClient.EXPECT().GetBucketInfo(gomock.Any()).DoAndReturn(
					func(_ string, _ ...oss.Option) (*oss.BucketInfo, error) {
						return &oss.BucketInfo{}, nil
					},
				)
			})

			Context("bucket lock need to be enabled", func() {
				BeforeEach(func() {
					ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
						func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
							return nil, oss.ServiceError{
								Code: "NoSuchWORMConfiguration",
							}
						},
					).AnyTimes()
				})

				It("should update bucket lock settings with unlocked retention policy", func() {
					backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
						Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 1, "locked": false }}`),
					}
					ossClient.EXPECT().CreateRetentionPolicy(gomock.Any(), gomock.Any()).Return("dummyWormID", nil)

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("should update bucket lock settings with locked retention policy", func() {
					backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
						Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 1, "locked": true }}`),
					}
					ossClient.EXPECT().CreateRetentionPolicy(gomock.Any(), gomock.Any()).Return("dummyWormID", nil)
					ossClient.EXPECT().LockRetentionPolicy(gomock.Any(), gomock.Any()).Return(nil)

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("should return error if updating bucket lock settings failed", func() {
					backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
						Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 1, "locked": false }}`),
					}
					ossClient.EXPECT().CreateRetentionPolicy(gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ string, _ int, _ ...oss.Option) (string, error) {
							return "", fmt.Errorf("unable to create retention policy on bucket")
						},
					)

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).Should(HaveOccurred())
				})

			})

			Context("bucket lock settings to be updated", func() {
				It("should lock the retention policy if wormConfigState=InProgress", func() {
					backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
						Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 1, "locked": true }}`),
					}
					ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
						func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
							return &oss.WormConfiguration{
								State:                 "InProgress",
								WormId:                "dummyID",
								RetentionPeriodInDays: 1,
							}, nil
						},
					).AnyTimes()
					ossClient.EXPECT().LockRetentionPolicy(gomock.Any(), gomock.Any()).Return(nil)

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("should retrun error if it failed to lock the retention policy if wormConfigState=InProgress", func() {
					backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
						Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 1, "locked": true }}`),
					}
					ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
						func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
							return &oss.WormConfiguration{
								State:                 "InProgress",
								WormId:                "dummyID",
								RetentionPeriodInDays: 1,
							}, nil
						},
					).AnyTimes()
					ossClient.EXPECT().LockRetentionPolicy(gomock.Any(), gomock.Any()).Return(fmt.Errorf("unable to lock the retention policy"))

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).Should(HaveOccurred())
				})

				It("should re-add the retentionPolicy if wormConfigState=Expired", func() {
					backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
						Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 1, "locked": true }}`),
					}
					ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
						func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
							return &oss.WormConfiguration{
								State:                 "Expired",
								WormId:                "dummyID",
								RetentionPeriodInDays: 1,
							}, nil
						},
					).AnyTimes()
					ossClient.EXPECT().AbortRetentionPolcy(gomock.Any()).Return(nil)
					ossClient.EXPECT().CreateRetentionPolicy(gomock.Any(), gomock.Any()).Return("dummyWormID", nil)
					ossClient.EXPECT().LockRetentionPolicy(gomock.Any(), gomock.Any()).Return(nil)

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("should return error if it failed to re-add the retentionPolicy when wormConfigState=Expired", func() {
					backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
						Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 1, "locked": true }}`),
					}
					ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
						func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
							return &oss.WormConfiguration{
								State:                 "Expired",
								WormId:                "dummyID",
								RetentionPeriodInDays: 1,
							}, nil
						},
					).AnyTimes()
					ossClient.EXPECT().AbortRetentionPolcy(gomock.Any()).Return(fmt.Errorf("unable to abort the retention policy"))

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).Should(HaveOccurred())
				})

				It("should extend the retentionPeriod when wormConfigState=Locked", func() {
					extenedRetentionPeriod := 2
					backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
						Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 2, "locked": true }}`),
					}
					ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
						func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
							return &oss.WormConfiguration{
								State:                 "Locked",
								WormId:                "dummyID",
								RetentionPeriodInDays: 1,
							}, nil
						},
					).AnyTimes()
					ossClient.EXPECT().UpdateRetentionPolicy(gomock.Any(), extenedRetentionPeriod, gomock.Any()).Return(nil)

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("should return error if it failed to extend the retentionPeriod when wormConfigState=Locked", func() {
					extenedRetentionPeriod := 2
					backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
						Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 2, "locked": true }}`),
					}
					ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
						func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
							return &oss.WormConfiguration{
								State:                 "Locked",
								WormId:                "dummyID",
								RetentionPeriodInDays: 1,
							}, nil
						},
					).AnyTimes()
					ossClient.EXPECT().UpdateRetentionPolicy(gomock.Any(), extenedRetentionPeriod, gomock.Any()).Return(fmt.Errorf("unable to extend the retentionPeriod"))

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).Should(HaveOccurred())
				})

			})

			Context("bucket lock need to be removed", func() {
				It("should return nil but bucket lock settings can't be removed as retention policy is in `Locked` state", func() {
					ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
						func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
							return &oss.WormConfiguration{
								State: "Locked",
							}, nil
						},
					)

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("should removed the retention policy as it's in `InProgress` state", func() {
					ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
						func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
							return &oss.WormConfiguration{
								State: "InProgress",
							}, nil
						},
					)
					ossClient.EXPECT().AbortRetentionPolcy(gomock.Any()).Return(nil)

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("should removed the retention policy as it's in `Expired` state", func() {
					ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
						func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
							return &oss.WormConfiguration{
								State: "Expired",
							}, nil
						},
					)
					ossClient.EXPECT().AbortRetentionPolcy(gomock.Any()).Return(nil)

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("should return error if it failed remove the retention policy", func() {
					ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
						func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
							return &oss.WormConfiguration{
								State: "InProgress",
							}, nil
						},
					)
					ossClient.EXPECT().AbortRetentionPolcy(gomock.Any()).Return(fmt.Errorf("unable to abort the retention policy"))

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).Should(HaveOccurred())
				})
			})

			Context("No update is required", func() {
				It("should do nothing when wormConfigState=Locked", func() {
					backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
						Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 2, "locked": true }}`),
					}
					ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
						func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
							return &oss.WormConfiguration{
								State:                 "Locked",
								RetentionPeriodInDays: 2,
							}, nil
						},
					).AnyTimes()

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("should do nothing when wormConfigState=InProgress", func() {
					backupBucket.Spec.ProviderConfig = &runtime.RawExtension{
						Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 1, "locked": false }}`),
					}
					ossClient.EXPECT().GetBucketWorm(gomock.Any()).DoAndReturn(
						func(_ string, _ ...oss.Option) (*oss.WormConfiguration, error) {
							return &oss.WormConfiguration{
								State:                 "InProgress",
								RetentionPeriodInDays: 1,
							}, nil
						},
					).AnyTimes()

					err := a.Reconcile(ctx, logger, backupBucket)
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
		})
	})

	Describe("#Delete", func() {
		var backupBucket *extensionsv1alpha1.BackupBucket

		BeforeEach(func() {
			backupBucket = &extensionsv1alpha1.BackupBucket{
				ObjectMeta: metav1.ObjectMeta{
					Name:      bucketName,
					Namespace: namespace,
				},
				Spec: extensionsv1alpha1.BackupBucketSpec{
					SecretRef: secretRef,
					Region:    region,
				},
			}

			c.EXPECT().Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
				func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
					*obj = *secret
					return nil
				},
			)
		})

		Context("when creation of alicloud's oss client fails", func() {
			It("should return an error", func() {
				alicloudClientFactory.EXPECT().NewOSSClient(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("failed to create alicloud oss client"))

				err := a.Delete(ctx, logger, backupBucket)
				Expect(err).Should(HaveOccurred())
			})
		})

		It("should delete the backup bucket successfully", func() {
			alicloudClientFactory.EXPECT().NewOSSClient(gomock.Any(), gomock.Any(), gomock.Any()).Return(ossClient, nil)
			ossClient.EXPECT().DeleteBucketIfExists(ctx, gomock.Any()).Return(nil)

			err := a.Delete(ctx, logger, backupBucket)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should return error if deletion of backup bucket fails", func() {
			alicloudClientFactory.EXPECT().NewOSSClient(gomock.Any(), gomock.Any(), gomock.Any()).Return(ossClient, nil)
			ossClient.EXPECT().DeleteBucketIfExists(ctx, gomock.Any()).Return(fmt.Errorf("failed to delete the backup bucket"))

			err := a.Delete(ctx, logger, backupBucket)
			Expect(err).Should(HaveOccurred())
		})
	})
})
