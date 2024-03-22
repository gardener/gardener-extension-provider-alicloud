// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"time"

	mockterraformer "github.com/gardener/gardener/extensions/pkg/terraformer/mock"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gardener/gardener-extension-provider-alicloud/imagevector"
)

var _ = Describe("Terraform", func() {
	var (
		ctrl *gomock.Controller

		purpose string
		infra   *extensionsv1alpha1.Infrastructure
		owner   *metav1.OwnerReference
		logger  = log.Log.WithName("test")
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		purpose = "purpose"

		infra = &extensionsv1alpha1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: extensionsv1alpha1.InfrastructureSpec{
				SecretRef: corev1.SecretReference{
					Name: "cloud",
				},
			},
		}
		owner = metav1.NewControllerRef(infra, extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.InfrastructureResource))
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#NewTerraformer", func() {
		It("should create a new terraformer", func() {
			var (
				factory                    = mockterraformer.NewMockFactory(ctrl)
				tf                         = mockterraformer.NewMockTerraformer(ctrl)
				config                     rest.Config
				disableProjectedTokenMount = false
			)

			gomock.InOrder(
				factory.EXPECT().
					NewForConfig(gomock.Any(), &config, purpose, infra.Namespace, infra.Name, imagevector.TerraformerImage()).
					Return(tf, nil),
				tf.EXPECT().UseProjectedTokenMount(!disableProjectedTokenMount).Return(tf),
				tf.EXPECT().SetLogLevel("info").Return(tf),
				tf.EXPECT().SetTerminationGracePeriodSeconds(int64(630)).Return(tf),
				tf.EXPECT().SetDeadlineCleaning(5*time.Minute).Return(tf),
				tf.EXPECT().SetDeadlinePod(15*time.Minute).Return(tf),
				tf.EXPECT().SetOwnerRef(owner).Return(tf),
			)

			actual, err := NewTerraformer(logger, factory, &config, purpose, infra, disableProjectedTokenMount)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(BeIdenticalTo(tf))
		})
	})

	Describe("#NewTerraformerWithAuth", func() {
		It("should create a new terraformer and initialize it with the credentials", func() {
			var (
				factory                    = mockterraformer.NewMockFactory(ctrl)
				tf                         = mockterraformer.NewMockTerraformer(ctrl)
				config                     rest.Config
				disableProjectedTokenMount = false
			)

			gomock.InOrder(
				factory.EXPECT().
					NewForConfig(gomock.Any(), &config, purpose, infra.Namespace, infra.Name, imagevector.TerraformerImage()).
					Return(tf, nil),
				tf.EXPECT().UseProjectedTokenMount(!disableProjectedTokenMount).Return(tf),
				tf.EXPECT().SetLogLevel("info").Return(tf),
				tf.EXPECT().SetTerminationGracePeriodSeconds(int64(630)).Return(tf),
				tf.EXPECT().SetDeadlineCleaning(5*time.Minute).Return(tf),
				tf.EXPECT().SetDeadlinePod(15*time.Minute).Return(tf),
				tf.EXPECT().SetOwnerRef(owner).Return(tf),
				tf.EXPECT().SetEnvVars(TerraformerEnvVars(infra.Spec.SecretRef)).Return(tf),
			)

			actual, err := NewTerraformerWithAuth(logger, factory, &config, purpose, infra, disableProjectedTokenMount)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(BeIdenticalTo(tf))
		})
	})

	Describe("#NewTerraformer", func() {
		It("should generate the correct env vars", func() {
			Expect(TerraformerEnvVars(infra.Spec.SecretRef)).To(ConsistOf(
				corev1.EnvVar{
					Name: "TF_VAR_ACCESS_KEY_ID",
					ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: infra.Spec.SecretRef.Name,
						},
						Key: "accessKeyID",
					}},
				},
				corev1.EnvVar{
					Name: "TF_VAR_ACCESS_KEY_SECRET",
					ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: infra.Spec.SecretRef.Name,
						},
						Key: "accessKeySecret",
					}},
				}))
		})
	})
})
