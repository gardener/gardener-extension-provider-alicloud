// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validator_test

import (
	"context"
	"strings"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/admission/validator"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
)

var _ = Describe("Secret validator", func() {
	var (
		ctx             = context.Background()
		secretValidator extensionswebhook.Validator
		secret          *corev1.Secret
	)

	BeforeEach(func() {
		secretValidator = validator.NewSecretValidator()
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "garden-test"},
			Data: map[string][]byte{
				alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16)),
				alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30)),
			},
		}
	})

	It("should reject incorrect object types", func() {
		Expect(secretValidator.Validate(ctx, &corev1.ConfigMap{}, nil)).To(MatchError("wrong object type *v1.ConfigMap"))
		Expect(secretValidator.Validate(ctx, secret, &corev1.ConfigMap{})).To(MatchError("wrong object type *v1.ConfigMap for old object"))
	})

	It("should accept valid credentials on create, metadata update, and rotation", func() {
		Expect(secretValidator.Validate(ctx, secret, nil)).To(Succeed())
		oldSecret := secret.DeepCopy()
		secret.Labels = map[string]string{"provider.shoot.gardener.cloud/alicloud": "true"}
		Expect(secretValidator.Validate(ctx, secret, oldSecret)).To(Succeed())
		oldSecret = secret.DeepCopy()
		secret.Data[alicloud.AccessKeySecret] = []byte(strings.Repeat("c", 30))
		Expect(secretValidator.Validate(ctx, secret, oldSecret)).To(Succeed())
	})

	DescribeTable("should reject credential injection on create and data-changing update",
		func(separator string) {
			oldSecret := secret.DeepCopy()
			secret.Data[alicloud.AccessKeySecret] = []byte(strings.Repeat("b", 30) + separator + "type = ecs_ram_role" + separator + "role_name =")
			Expect(secretValidator.Validate(ctx, secret, nil)).To(MatchError(ContainSubstring("must not contain control characters")))
			Expect(secretValidator.Validate(ctx, secret, oldSecret)).To(MatchError(ContainSubstring("must not contain control characters")))
		},
		Entry("LF", "\n"),
		Entry("CR", "\r"),
		Entry("CRLF", "\r\n"),
	)

	DescribeTable("should revalidate unchanged invalid credentials on metadata-only updates",
		func(alreadyLabeled bool, value string) {
			secret.Data[alicloud.AccessKeySecret] = []byte(value)
			if alreadyLabeled {
				secret.Labels = map[string]string{"provider.shoot.gardener.cloud/alicloud": "true"}
			}
			oldSecret := secret.DeepCopy()
			secret.Labels = map[string]string{"provider.shoot.gardener.cloud/alicloud": "true"}
			secret.Annotations = map[string]string{"test": "metadata-only-update"}
			Expect(secretValidator.Validate(ctx, secret, oldSecret)).To(HaveOccurred())
		},
		Entry("label added to an injected secret", false, strings.Repeat("b", 30)+"\ntype = ecs_ram_role\nrole_name ="),
		Entry("already labeled injected secret", true, strings.Repeat("b", 30)+"\ntype = ecs_ram_role\nrole_name ="),
		Entry("label added to a short secret", false, "too-short"),
		Entry("already labeled short secret", true, "too-short"),
	)

	It("should allow replacing previously invalid credentials with valid credentials", func() {
		oldSecret := secret.DeepCopy()
		oldSecret.Data[alicloud.AccessKeySecret] = []byte(strings.Repeat("b", 30) + "\ntype = ecs_ram_role\nrole_name =")
		Expect(secretValidator.Validate(ctx, secret, oldSecret)).To(Succeed())
	})
})
