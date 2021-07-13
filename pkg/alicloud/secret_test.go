// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package alicloud

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

const (
	accessKeyID     = "accessKeyID"
	accessKeySecret = "accessKeySecret"
)

var _ = Describe("Alicloud Suite", func() {
	Describe("#ReadSecretCredentials", func() {
		Context("DNS keys are not allowed", func() {
			It("should correctly read the credentials if non-DNS keys are used", func() {
				creds, err := ReadSecretCredentials(&corev1.Secret{
					Data: map[string][]byte{
						AccessKeyID:     []byte(accessKeyID),
						AccessKeySecret: []byte(accessKeySecret),
					},
				}, false)

				Expect(err).NotTo(HaveOccurred())
				Expect(creds).To(Equal(&Credentials{
					AccessKeyID:     accessKeyID,
					AccessKeySecret: accessKeySecret,
				}))
			})

			It("should fail if DNS keys are used", func() {
				_, err := ReadSecretCredentials(&corev1.Secret{
					Data: map[string][]byte{
						DNSAccessKeyID:     []byte(accessKeyID),
						DNSAccessKeySecret: []byte(accessKeySecret),
					},
				}, false)

				Expect(err).To(HaveOccurred())
			})
		})

		Context("DNS keys are allowed", func() {
			It("should correctly read the credentials if non-DNS keys are used", func() {
				creds, err := ReadSecretCredentials(&corev1.Secret{
					Data: map[string][]byte{
						AccessKeyID:     []byte(accessKeyID),
						AccessKeySecret: []byte(accessKeySecret),
					},
				}, true)

				Expect(err).NotTo(HaveOccurred())
				Expect(creds).To(Equal(&Credentials{
					AccessKeyID:     accessKeyID,
					AccessKeySecret: accessKeySecret,
				}))
			})

			It("should correctly read the credentials if DNS keys are used", func() {
				creds, err := ReadSecretCredentials(&corev1.Secret{
					Data: map[string][]byte{
						DNSAccessKeyID:     []byte(accessKeyID),
						DNSAccessKeySecret: []byte(accessKeySecret),
					},
				}, true)

				Expect(err).NotTo(HaveOccurred())
				Expect(creds).To(Equal(&Credentials{
					AccessKeyID:     accessKeyID,
					AccessKeySecret: accessKeySecret,
				}))
			})
		})

		It("should fail if the data section is nil", func() {
			_, err := ReadSecretCredentials(&corev1.Secret{}, false)
			Expect(err).To(HaveOccurred())
		})

		It("should fail if access key id is missing", func() {
			_, err := ReadSecretCredentials(&corev1.Secret{
				Data: map[string][]byte{
					AccessKeySecret: []byte(accessKeySecret),
				},
			}, false)

			Expect(err).To(HaveOccurred())
		})

		It("should fail if access key secret is missing", func() {
			_, err := ReadSecretCredentials(&corev1.Secret{
				Data: map[string][]byte{
					AccessKeyID: []byte(accessKeyID),
				},
			}, false)

			Expect(err).To(HaveOccurred())
		})
	})
})
