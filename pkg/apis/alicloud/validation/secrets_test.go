// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"
)

var _ = Describe("Secret validation", func() {

	DescribeTable("#ValidateCloudProviderSecret",
		func(data map[string][]byte, matcher gomegatypes.GomegaMatcher) {
			secret := &corev1.Secret{
				Data: data,
			}
			err := ValidateCloudProviderSecret(secret)

			Expect(err).To(matcher)
		},

		Entry("should return error when the access key field is missing",
			map[string][]byte{
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30)),
			},
			HaveOccurred(),
		),

		Entry("should return error when the access key is empty",
			map[string][]byte{
				alicloud.AccessKeyID:     {},
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30)),
			},
			HaveOccurred(),
		),

		Entry("should return error when the access key is too short",
			map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 15)),
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30)),
			},
			HaveOccurred(),
		),

		Entry("should return error when the access key is too long",
			map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 129)),
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30)),
			},
			HaveOccurred(),
		),

		Entry("should return error when the access key does not contain only alphanumeric characters and [._=]",
			map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16) + " "),
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30)),
			},
			HaveOccurred(),
		),

		Entry("should return error when the secret access key field is missing",
			map[string][]byte{
				alicloud.AccessKeyID: []byte(strings.Repeat("a", 16)),
			},
			HaveOccurred(),
		),

		Entry("should return error when the secret access key is empty",
			map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16)),
				alicloud.AccessKeySecret: {},
			},
			HaveOccurred(),
		),

		Entry("should return error when the secret access key is too short",
			map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16)),
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 29)),
			},
			HaveOccurred(),
		),

		Entry("should return error when the secret access key contains a trailing new line",
			map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16)),
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30) + "\n"),
			},
			HaveOccurred(),
		),

		Entry("should succeed when the client credentials are valid (shortest possilble access key)",
			map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16)),
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30)),
			},
			BeNil(),
		),

		Entry("should succeed when the client credentials are valid (longest possilble access key)",
			map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 128)),
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30)),
			},
			BeNil(),
		),
	)
})
