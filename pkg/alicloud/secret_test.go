// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	. "github.com/onsi/ginkgo/v2"
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
						dnsAccessKeyID:     []byte(accessKeyID),
						dnsAccessKeySecret: []byte(accessKeySecret),
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
						dnsAccessKeyID:     []byte(accessKeyID),
						dnsAccessKeySecret: []byte(accessKeySecret),
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
