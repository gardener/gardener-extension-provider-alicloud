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
			secret.Labels = map[string]string{"provider.shoot.gardener.cloud/alicloud": "true"}
			oldSecret := secret.DeepCopy()
			secret.Data[alicloud.AccessKeySecret] = []byte(strings.Repeat("b", 30) + separator + "type = ecs_ram_role" + separator + "role_name =")
			Expect(secretValidator.Validate(ctx, secret, nil)).To(MatchError(ContainSubstring("must not contain control characters")))
			Expect(secretValidator.Validate(ctx, secret, oldSecret)).To(MatchError(ContainSubstring("must not contain control characters")))
		},
		Entry("LF", "\n"),
		Entry("CR", "\r"),
		Entry("CRLF", "\r\n"),
	)

	It("should reject an oversized secret access key on create and data-changing update", func() {
		secret.Labels = map[string]string{"provider.shoot.gardener.cloud/alicloud": "true"}
		oldSecret := secret.DeepCopy()
		secret.Data[alicloud.AccessKeySecret] = []byte(strings.Repeat("b", 129))
		Expect(secretValidator.Validate(ctx, secret, nil)).To(MatchError(ContainSubstring("cannot be longer than 128 bytes")))
		Expect(secretValidator.Validate(ctx, secret, oldSecret)).To(MatchError(ContainSubstring("cannot be longer than 128 bytes")))
	})

	DescribeTable("should validate unchanged invalid credentials when the provider label is added or enabled",
		func(oldLabels map[string]string, value string) {
			secret.Data[alicloud.AccessKeySecret] = []byte(value)
			secret.Labels = oldLabels
			oldSecret := secret.DeepCopy()
			secret.Labels = map[string]string{"provider.shoot.gardener.cloud/alicloud": "true"}
			secret.Annotations = map[string]string{"test": "metadata-only-update"}
			Expect(secretValidator.Validate(ctx, secret, oldSecret)).To(HaveOccurred())
		},
		Entry("label added to an injected secret", nil, strings.Repeat("b", 30)+"\ntype = ecs_ram_role\nrole_name ="),
		Entry("label added to a short secret", nil, "too-short"),
		Entry("label added to an oversized secret", nil, strings.Repeat("b", 129)),
		Entry("label changed from false to true", map[string]string{"provider.shoot.gardener.cloud/alicloud": "false"}, "too-short"),
		Entry("label changed from empty to true", map[string]string{"provider.shoot.gardener.cloud/alicloud": ""}, "too-short"),
		Entry("only extension provider label present", map[string]string{"provider.extensions.gardener.cloud/alicloud": "true"}, "too-short"),
	)

	DescribeTable("should allow metadata-only updates to already provider-labeled secrets with unchanged invalid credentials",
		func(value string, removeLabel bool) {
			secret.Data[alicloud.AccessKeySecret] = []byte(value)
			secret.Labels = map[string]string{"provider.shoot.gardener.cloud/alicloud": "true"}
			oldSecret := secret.DeepCopy()
			secret.Annotations = map[string]string{"test": "metadata-only-update"}
			if removeLabel {
				delete(secret.Labels, "provider.shoot.gardener.cloud/alicloud")
			}
			Expect(secretValidator.Validate(ctx, secret, oldSecret)).To(Succeed())
		},
		Entry("annotation added to an injected secret", strings.Repeat("b", 30)+"\ntype = ecs_ram_role\nrole_name =", false),
		Entry("annotation added to a short secret", "too-short", false),
		Entry("annotation added to an oversized secret", strings.Repeat("b", 129), false),
		Entry("provider label removed from an injected secret", strings.Repeat("b", 30)+"\ntype = ecs_ram_role\nrole_name =", true),
		Entry("provider label removed from a short secret", "too-short", true),
		Entry("provider label removed from an oversized secret", strings.Repeat("b", 129), true),
	)

	It("should allow replacing previously invalid credentials with valid credentials", func() {
		oldSecret := secret.DeepCopy()
		oldSecret.Data[alicloud.AccessKeySecret] = []byte(strings.Repeat("b", 30) + "\ntype = ecs_ram_role\nrole_name =")
		Expect(secretValidator.Validate(ctx, secret, oldSecret)).To(Succeed())
	})
})
