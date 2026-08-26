// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validator_test

import (
	"context"
	"strings"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	testutils "github.com/gardener/gardener/pkg/utils/test"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/admission/validator"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
)

var _ = Describe("SecretBinding validator", func() {

	Describe("#Validate", func() {
		const (
			namespace = "garden-dev"
			name      = "my-provider-account"
		)

		var (
			secretBindingValidator extensionswebhook.Validator

			ctx           = context.TODO()
			secretBinding = &core.SecretBinding{
				Provider: &core.SecretBindingProvider{
					Type: alicloud.Type,
				},
				SecretRef: corev1.SecretReference{
					Name:      name,
					Namespace: namespace,
				},
			}

			scheme = runtime.NewScheme()
			_      = corev1.AddToScheme(scheme)
		)

		newValidator := func(apiReaderObjects ...client.Object) extensionswebhook.Validator {
			rb := fakeclient.NewClientBuilder().WithScheme(scheme)
			if len(apiReaderObjects) > 0 {
				rb = rb.WithObjects(apiReaderObjects...)
			}
			mgr := &testutils.FakeManager{APIReader: rb.Build()}
			return validator.NewSecretBindingValidator(mgr)
		}

		It("should return err when obj is not a SecretBinding", func() {
			secretBindingValidator = newValidator()
			err := secretBindingValidator.Validate(ctx, &corev1.Secret{}, nil)
			Expect(err).To(MatchError("wrong object type *v1.Secret"))
		})

		It("should return err when oldObj is not a SecretBinding", func() {
			secretBindingValidator = newValidator()
			err := secretBindingValidator.Validate(ctx, &core.SecretBinding{}, &corev1.Secret{})
			Expect(err).To(MatchError("wrong object type *v1.Secret for old object"))
		})

		It("should return err if it fails to get the corresponding Secret", func() {
			secretBindingValidator = newValidator() // no secret pre-loaded → NotFound
			err := secretBindingValidator.Validate(ctx, secretBinding, nil)
			Expect(err).To(HaveOccurred())
		})

		It("should return err when the corresponding Secret is not valid", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Data:       map[string][]byte{"foo": []byte("bar")},
			}
			secretBindingValidator = newValidator(secret)
			err := secretBindingValidator.Validate(ctx, secretBinding, nil)
			Expect(err).To(HaveOccurred())
		})

		It("should return nil when the corresponding Secret is valid", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Data: map[string][]byte{
					alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16)),
					alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30)),
				},
			}
			secretBindingValidator = newValidator(secret)
			err := secretBindingValidator.Validate(ctx, secretBinding, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return nil when the provider type did not change", func() {
			secretBindingValidator = newValidator()
			old := secretBinding.DeepCopy()
			err := secretBindingValidator.Validate(ctx, secretBinding, old)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return err when the provider type changed (to alicloud) and the corresponding Secret is not valid", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Data:       map[string][]byte{"foo": []byte("bar")},
			}
			secretBindingValidator = newValidator(secret)
			old := secretBinding.DeepCopy()
			old.Provider = nil
			err := secretBindingValidator.Validate(ctx, secretBinding, old)
			Expect(err).To(HaveOccurred())
		})
	})
})
