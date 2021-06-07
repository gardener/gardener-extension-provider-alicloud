// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"context"
	"time"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client/ros"
	mockalicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/mock/provider-alicloud/alicloud/client"
	gcorev1alph1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gcorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"
)

var _ = Describe("Encrypt Image tools", func() {
	DescribeTable("#UseEncryptedSystemDisk",
		func(volume interface{}, expectEncrypted, expectErr bool) {
			encrypted, err := UseEncryptedSystemDisk(volume)
			expectResults(encrypted, expectEncrypted, err, expectErr)
		},
		Entry("nil input doesn't use encryption", nil, false, false),
		Entry("corev1beta1 image with nil Encryption value doesn't use encryption", &gcorev1beta1.Volume{}, false, false),
		Entry("corev1beta1 image with false Encryption value doesn't use encryption", &gcorev1beta1.Volume{Encrypted: pointer.BoolPtr(false)}, false, false),
		Entry("corev1beta1 image with true Encryption value uses encryption", &gcorev1beta1.Volume{Encrypted: pointer.BoolPtr(true)}, true, false),
		Entry("extensionsv1alpha1 image with nil Encryption value doesn't use encryption", &gcorev1beta1.Volume{}, false, false),
		Entry("extensionsv1alpha1 image with false Encryption value doesn't use encryption", &gcorev1beta1.Volume{Encrypted: pointer.BoolPtr(false)}, false, false),
		Entry("extensionsv1alpha1 image with true Encryption value uses encryption", &gcorev1beta1.Volume{Encrypted: pointer.BoolPtr(true)}, true, false),
		Entry("unsupported type of image returns error", &gcorev1alph1.Volume{}, true, true),
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
				Expect(GetEncryptImageStackName(imageName, imageVersion)).To(Equal("encrypt_image_GardenLinux_1-184-0"))
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
