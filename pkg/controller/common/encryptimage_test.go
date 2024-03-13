// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"time"

	gcorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/pointer"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client/ros"
	mockalicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/mock/provider-alicloud/alicloud/client"
)

var _ = Describe("Encrypt Image tools", func() {
	DescribeTable("#UseEncryptedSystemDisk",
		func(volume interface{}, expectEncrypted, expectErr bool) {
			encrypted, err := UseEncryptedSystemDisk(volume)
			expectResults(encrypted, expectEncrypted, err, expectErr)
		},
		Entry("nil input doesn't use encryption", nil, false, false),
		Entry("corev1beta1 image with nil Encryption value doesn't use encryption", &gcorev1beta1.Volume{}, false, false),
		Entry("corev1beta1 image with false Encryption value doesn't use encryption", &gcorev1beta1.Volume{Encrypted: pointer.Bool(false)}, false, false),
		Entry("corev1beta1 image with true Encryption value uses encryption", &gcorev1beta1.Volume{Encrypted: pointer.Bool(true)}, true, false),
		Entry("extensionsv1alpha1 image with nil Encryption value doesn't use encryption", &gcorev1beta1.Volume{}, false, false),
		Entry("extensionsv1alpha1 image with false Encryption value doesn't use encryption", &gcorev1beta1.Volume{Encrypted: pointer.Bool(false)}, false, false),
		Entry("extensionsv1alpha1 image with true Encryption value uses encryption", &gcorev1beta1.Volume{Encrypted: pointer.Bool(true)}, true, false),
	)

	Context("#ImageEncrypter", func() {
		var (
			ctrl             *gomock.Controller
			regionID         = "cn-shanghai"
			sourceImageID    = "m-123456"
			imageName        = "GardenLinux"
			imageVersion     = "1.184.0"
			stackID          = "abcd-efgh-1234"
			encryptedImgID   = "m-234567"
			defaultEncryptor *imageEncryptor
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			var ok bool
			defaultEncryptor, ok = NewImageEncryptor(nil,
				regionID,
				imageName,
				imageVersion,
				sourceImageID,
			).(*imageEncryptor)
			Expect(ok).To(Equal(true))
		})
		AfterEach(func() {
			ctrl.Finish()
		})

		Describe("#GetEncryptImageStackName", func() {
			It("should compose correct name", func() {
				Expect(GetEncryptImageStackName(imageName, imageVersion, regionID)).To(Equal("encrypt_image_GardenLinux_1-184-0_cn-shanghai"))
			})
		})

		Describe("#TryToGetEncryptedImageID", func() {
			var (
				shootROSClient *mockalicloudclient.MockROS
				ctx            context.Context
				timeout        = 30 * time.Millisecond
				interval       = 1 * time.Millisecond
			)

			BeforeEach(func() {
				ctx = context.TODO()
				shootROSClient = mockalicloudclient.NewMockROS(ctrl)
			})

			It("should succeed when a successful Stack already exists", func() {
				listResponse := ros.ListStacksResponse{
					Stacks: []ros.Stack{
						{
							StackId: stackID,
						},
					},
				}
				getResponse := ros.GetStackResponse{
					Status:  "CREATE_COMPLETE",
					Outputs: []map[string]string{{"OutputKey": "ImageId", "OutputValue": encryptedImgID}},
				}
				shootROSClient.EXPECT().ListStacks(gomock.Any()).Return(&listResponse, nil)
				shootROSClient.EXPECT().GetStack(gomock.Any()).Return(&getResponse, nil)
				defaultEncryptor.rosClient = shootROSClient
				result, err := defaultEncryptor.TryToGetEncryptedImageID(ctx, timeout, interval)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(encryptedImgID))
			})
			It("should fail if existing Stack failed", func() {
				listResponse := ros.ListStacksResponse{
					Stacks: []ros.Stack{
						{
							StackId: stackID,
						},
					},
				}
				getResponse := ros.GetStackResponse{
					Status:  "CREATE_FAILED",
					Outputs: []map[string]string{{"OutputKey": "ImageId", "OutputValue": encryptedImgID}},
				}
				shootROSClient.EXPECT().ListStacks(gomock.Any()).Return(&listResponse, nil)
				shootROSClient.EXPECT().GetStack(gomock.Any()).Return(&getResponse, nil)
				defaultEncryptor.rosClient = shootROSClient
				_, err := defaultEncryptor.TryToGetEncryptedImageID(ctx, timeout, interval)
				Expect(err).To(HaveOccurred())
			})
			It("should succeed when no stack exists and a stack is created successfully", func() {
				listResponse := ros.ListStacksResponse{}
				getResponse := ros.GetStackResponse{
					Status:  "CREATE_COMPLETE",
					Outputs: []map[string]string{{"OutputKey": "ImageId", "OutputValue": encryptedImgID}},
				}
				createResponse := ros.CreateStackResponse{
					StackId: stackID,
				}
				shootROSClient.EXPECT().ListStacks(gomock.Any()).Return(&listResponse, nil)
				shootROSClient.EXPECT().GetStack(gomock.Any()).Return(&getResponse, nil)
				shootROSClient.EXPECT().CreateStack(gomock.Any()).Return(&createResponse, nil)
				defaultEncryptor.rosClient = shootROSClient
				result, err := defaultEncryptor.TryToGetEncryptedImageID(ctx, timeout, interval)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(encryptedImgID))
			})
			It("should succeed when no stack exists and a stack is created successfully", func() {
				listResponse := ros.ListStacksResponse{}
				createResponse := ros.CreateStackResponse{
					StackId: stackID,
				}
				getResponse := ros.GetStackResponse{
					Status:  "CREATE_COMPLETE",
					Outputs: []map[string]string{{"OutputKey": "ImageId", "OutputValue": encryptedImgID}},
				}
				shootROSClient.EXPECT().ListStacks(gomock.Any()).Return(&listResponse, nil)
				shootROSClient.EXPECT().GetStack(gomock.Any()).Return(&getResponse, nil)
				shootROSClient.EXPECT().CreateStack(gomock.Any()).Return(&createResponse, nil)
				defaultEncryptor.rosClient = shootROSClient
				result, err := defaultEncryptor.TryToGetEncryptedImageID(ctx, timeout, interval)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(encryptedImgID))
			})
			It("should fail when no stack exists and a stack is created with failure", func() {
				listResponse := ros.ListStacksResponse{}
				createResponse := ros.CreateStackResponse{
					StackId: stackID,
				}
				getResponse := ros.GetStackResponse{
					Status:  "CREATE_FAILED",
					Outputs: []map[string]string{{"OutputKey": "ImageId", "OutputValue": encryptedImgID}},
				}
				shootROSClient.EXPECT().ListStacks(gomock.Any()).Return(&listResponse, nil)
				shootROSClient.EXPECT().GetStack(gomock.Any()).Return(&getResponse, nil)
				shootROSClient.EXPECT().CreateStack(gomock.Any()).Return(&createResponse, nil)
				defaultEncryptor.rosClient = shootROSClient
				_, err := defaultEncryptor.TryToGetEncryptedImageID(ctx, timeout, interval)
				Expect(err).To(HaveOccurred())
			})
			It("should fail when no stack exists and a stack is created with timout", func() {
				listResponse := ros.ListStacksResponse{}
				createResponse := ros.CreateStackResponse{
					StackId: stackID,
				}
				getResponse := ros.GetStackResponse{
					Status:  "CREATE_INPROCESS",
					Outputs: []map[string]string{{"OutputKey": "ImageId", "OutputValue": encryptedImgID}},
				}
				shootROSClient.EXPECT().ListStacks(gomock.Any()).Return(&listResponse, nil)
				shootROSClient.EXPECT().GetStack(gomock.Any()).AnyTimes().Return(&getResponse, nil)
				shootROSClient.EXPECT().CreateStack(gomock.Any()).Return(&createResponse, nil)
				defaultEncryptor.rosClient = shootROSClient
				_, err := defaultEncryptor.TryToGetEncryptedImageID(ctx, timeout, interval)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

func expectResults(result, expected bool, err error, expectErr bool) {
	if !expectErr {
		Expect(result).To(Equal(expected))
		Expect(err).NotTo(HaveOccurred())
	} else {
		Expect(err).To(HaveOccurred())
	}
}
