// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validator_test

import (
	"context"
	"errors"
	"strings"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/security"
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

var _ = Describe("CredentialsBinding validator", func() {
	Describe("#Validate", func() {
		const (
			namespace = "garden-dev"
			name      = "my-provider-account"
		)

		var (
			credentialsBindingValidator extensionswebhook.Validator

			ctx                = context.TODO()
			credentialsBinding *security.CredentialsBinding

			scheme = runtime.NewScheme()
			_      = corev1.AddToScheme(scheme)
		)

		newValidator := func(apiReaderObjects ...client.Object) extensionswebhook.Validator {
			rb := fakeclient.NewClientBuilder().WithScheme(scheme)
			if len(apiReaderObjects) > 0 {
				rb = rb.WithObjects(apiReaderObjects...)
			}
			mgr := &testutils.FakeManager{APIReader: rb.Build()}
			return validator.NewCredentialsBindingValidator(mgr)
		}

		BeforeEach(func() {
			credentialsBinding = &security.CredentialsBinding{
				CredentialsRef: corev1.ObjectReference{
					Name:       name,
					Namespace:  namespace,
					Kind:       "Secret",
					APIVersion: "v1",
				},
			}
		})

		It("should return err when obj is not a CredentialsBinding", func() {
			credentialsBindingValidator = newValidator()
			err := credentialsBindingValidator.Validate(ctx, &corev1.Secret{}, nil)
			Expect(err).To(MatchError("wrong object type *v1.Secret"))
		})

		It("should return err when oldObj is not a CredentialsBinding", func() {
			credentialsBindingValidator = newValidator()
			err := credentialsBindingValidator.Validate(ctx, &security.CredentialsBinding{}, &corev1.Secret{})
			Expect(err).To(MatchError("wrong object type *v1.Secret for old object"))
		})

		It("should return err if the CredentialsBinding references unknown credentials type", func() {
			credentialsBindingValidator = newValidator()
			credentialsBinding.CredentialsRef.APIVersion = "unknown"
			err := credentialsBindingValidator.Validate(ctx, credentialsBinding, nil)
			Expect(err).To(MatchError(errors.New(`unsupported credentials reference: version "unknown", kind "Secret"`)))
		})

		It("should return err if it fails to get the corresponding Secret", func() {
			credentialsBindingValidator = newValidator() // no secret pre-loaded → NotFound
			err := credentialsBindingValidator.Validate(ctx, credentialsBinding, nil)
			Expect(err).To(HaveOccurred())
		})

		It("should return err when the corresponding Secret is not valid", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Data:       map[string][]byte{"foo": []byte("bar")},
			}
			credentialsBindingValidator = newValidator(secret)
			err := credentialsBindingValidator.Validate(ctx, credentialsBinding, nil)
			Expect(err).To(HaveOccurred())
		})

		It("should succeed when the corresponding Secret is valid", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Data: map[string][]byte{
					alicloud.AccessKeyID:     []byte(strings.Repeat("a", 16)),
					alicloud.AccessKeySecret: []byte(strings.Repeat("b", 30)),
				},
			}
			credentialsBindingValidator = newValidator(secret)
			Expect(credentialsBindingValidator.Validate(ctx, credentialsBinding, nil)).To(Succeed())
		})

		It("should return nil when the CredentialsBinding did not change", func() {
			credentialsBindingValidator = newValidator()
			old := credentialsBinding.DeepCopy()
			Expect(credentialsBindingValidator.Validate(ctx, credentialsBinding, old)).To(Succeed())
		})
	})
})
