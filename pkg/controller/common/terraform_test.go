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

package common

import (
	"time"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	mockterraformer "github.com/gardener/gardener/pkg/mock/gardener/extensions/terraformer"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/imagevector"
)

var _ = Describe("Terraform", func() {
	var (
		ctrl *gomock.Controller

		purpose string
		infra   *extensionsv1alpha1.Infrastructure
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
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#NewTerraformer", func() {
		It("should create a new terraformer", func() {
			var (
				factory = mockterraformer.NewMockFactory(ctrl)
				tf      = mockterraformer.NewMockTerraformer(ctrl)
				config  rest.Config
			)

			gomock.InOrder(
				factory.EXPECT().
					NewForConfig(gomock.Any(), &config, purpose, infra.Namespace, infra.Name, imagevector.TerraformerImage()).
					Return(tf, nil),
				tf.EXPECT().UseV2(true).Return(tf),
				tf.EXPECT().SetLogLevel("info").Return(tf),
				tf.EXPECT().SetTerminationGracePeriodSeconds(int64(630)).Return(tf),
				tf.EXPECT().SetDeadlineCleaning(5*time.Minute).Return(tf),
				tf.EXPECT().SetDeadlinePod(15*time.Minute).Return(tf),
			)

			actual, err := NewTerraformer(logger, factory, &config, purpose, infra)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(BeIdenticalTo(tf))
		})
	})

	Describe("#NewTerraformerWithAuth", func() {
		It("should create a new terraformer and initialize it with the credentials", func() {
			var (
				factory = mockterraformer.NewMockFactory(ctrl)
				tf      = mockterraformer.NewMockTerraformer(ctrl)
				config  rest.Config
			)

			gomock.InOrder(
				factory.EXPECT().
					NewForConfig(gomock.Any(), &config, purpose, infra.Namespace, infra.Name, imagevector.TerraformerImage()).
					Return(tf, nil),
				tf.EXPECT().UseV2(true).Return(tf),
				tf.EXPECT().SetLogLevel("info").Return(tf),
				tf.EXPECT().SetTerminationGracePeriodSeconds(int64(630)).Return(tf),
				tf.EXPECT().SetDeadlineCleaning(5*time.Minute).Return(tf),
				tf.EXPECT().SetDeadlinePod(15*time.Minute).Return(tf),
				tf.EXPECT().SetEnvVars(TerraformerEnvVars(infra.Spec.SecretRef)).Return(tf),
			)

			actual, err := NewTerraformerWithAuth(logger, factory, &config, purpose, infra)
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
