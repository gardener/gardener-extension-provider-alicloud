// SPDX-FileCopyrightText: Contributors to the Gardener project
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

		Entry("should return error when the secret access key contains a new line",
			map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16)),
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30) + "\n"),
			},
			HaveOccurred(),
		),

		Entry("should return error when the secret access key contains an embedded new line",
			map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16)),
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 15) + "\n" + strings.Repeat("b", 15)),
			},
			HaveOccurred(),
		),

		Entry("should succeed when the client credentials are valid (shortest possible access key)",
			map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16)),
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30)),
			},
			BeNil(),
		),

		Entry("should succeed when the client credentials are valid (longest possible access key)",
			map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 128)),
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30)),
			},
			BeNil(),
		),

		Entry("should accept supported punctuation in credential values",
			map[string][]byte{
				alicloud.AccessKeyID:     []byte("ABcd0123._=efghiJ"),
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30) + "/+=._-"),
			},
			BeNil(),
		),
	)

	DescribeTable("should enforce the secret access key maximum length in bytes without exposing credentials",
		func(value string, tooLong bool) {
			secret := &corev1.Secret{Data: map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16)),
				alicloud.AccessKeySecret: []byte(value),
			}}
			secret.Namespace = "garden-test"
			secret.Name = "credentials"

			err := ValidateCloudProviderSecret(secret)
			if tooLong {
				Expect(err).To(MatchError(`field "accessKeySecret" in secret garden-test/credentials cannot be longer than 128 bytes`))
				Expect(err.Error()).NotTo(ContainSubstring(value))
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("one byte below the maximum", strings.Repeat("b", 127), false),
		Entry("exactly the maximum", strings.Repeat("b", 128), false),
		Entry("one byte above the maximum", strings.Repeat("b", 129), true),
		Entry("multibyte characters exactly at the maximum", strings.Repeat("é", 64), false),
		Entry("multibyte characters exceeding the maximum", strings.Repeat("é", 64)+"b", true),
	)

	It("should reject every control character and line separator anywhere in the secret access key", func() {
		characters := []rune{'\u2028', '\u2029'}
		for r := rune(0); r <= '\u009f'; r++ {
			if r < ' ' || r >= '\u007f' {
				characters = append(characters, r)
			}
		}

		for _, character := range characters {
			for _, offset := range []int{0, 15, 30} {
				secret := &corev1.Secret{Data: map[string][]byte{
					alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16)),
					alicloud.AccessKeySecret: []byte(strings.Repeat("b", offset) + string(character) + strings.Repeat("b", 30-offset)),
				}}
				Expect(ValidateCloudProviderSecret(secret)).To(MatchError(ContainSubstring("must not contain control characters or line separators")), "character U+%04X at offset %d", character, offset)
			}
		}
	})

	DescribeTable("should reject INI injection in the secret access key without exposing credentials",
		func(injection string) {
			value := strings.Repeat("b", 30) + injection
			secret := &corev1.Secret{Data: map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16)),
				alicloud.AccessKeySecret: []byte(value),
			}}
			err := ValidateCloudProviderSecret(secret)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).NotTo(ContainSubstring(value))
		},
		Entry("LF RAM role override", "\ntype = ecs_ram_role\nrole_name ="),
		Entry("CR RAM role override", "\rtype = ecs_ram_role\rrole_name ="),
		Entry("CRLF RAM role override", "\r\ntype = ecs_ram_role\r\nrole_name ="),
		Entry("duplicate default section", "\n[default]\ntype = ecs_ram_role\nrole_name ="),
		Entry("hash comment", "#comment"),
		Entry("semicolon comment", ";comment"),
	)
})
