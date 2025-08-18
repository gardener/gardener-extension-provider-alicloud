// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/webhook"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/config"
)

var _ = Describe("Mutator", func() {
	var (
		mutator         webhook.Mutator
		serviceConfig   = &config.Service{BackendLoadBalancerSpec: "slb.s1.small"}
		nginxIngressSvc *corev1.Service
	)

	BeforeEach(func() {
		mutator = NewMutator(serviceConfig)

		nginxIngressSvc = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "addons-nginx-ingress-controller",
				Namespace: metav1.NamespaceSystem,
			},
			Spec: corev1.ServiceSpec{
				Type:                  corev1.ServiceTypeLoadBalancer,
				ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
			},
		}
	})

	Describe("#Mutate", func() {
		It("should set ExternalTrafficPolicy to Local", func() {
			err := mutator.Mutate(context.TODO(), nginxIngressSvc, nil)

			Expect(err).To(Not(HaveOccurred()))
			Expect(nginxIngressSvc.Spec.ExternalTrafficPolicy).To(Equal(corev1.ServiceExternalTrafficPolicyTypeLocal))
		})

		It("should not overwrite .spec.healthCheckNodePort", func() {
			oldNginxIngressSvc := nginxIngressSvc.DeepCopy()
			oldNginxIngressSvc.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeLocal
			oldNginxIngressSvc.Spec.HealthCheckNodePort = 31280

			err := mutator.Mutate(context.TODO(), nginxIngressSvc, oldNginxIngressSvc)

			Expect(err).To(Not(HaveOccurred()))
			Expect(oldNginxIngressSvc.Spec.ExternalTrafficPolicy).To(Equal(corev1.ServiceExternalTrafficPolicyTypeLocal))
			Expect(oldNginxIngressSvc.Spec.HealthCheckNodePort).To(Equal(int32(31280)))
		})
	})
})
