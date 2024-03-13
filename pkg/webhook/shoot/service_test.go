// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
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
		vpnSvc          *corev1.Service
		nginxIngressSvc *corev1.Service
		otherSvc        *corev1.Service
	)

	BeforeEach(func() {
		mutator = NewMutator(serviceConfig)

		vpnSvc = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vpn-shoot",
				Namespace: metav1.NamespaceSystem,
			},
			Spec: corev1.ServiceSpec{
				Type:                  corev1.ServiceTypeLoadBalancer,
				ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
			},
		}
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
		otherSvc = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other",
				Namespace: metav1.NamespaceSystem,
			},
			Spec: corev1.ServiceSpec{ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster},
		}
	})

	Describe("#MutateLBService", func() {
		It("should set ExternalTrafficPolicy to Local for vpn-shoot service", func() {
			err := mutator.Mutate(context.TODO(), vpnSvc, nil)

			Expect(err).To(Not(HaveOccurred()))
			Expect(vpnSvc.Spec.ExternalTrafficPolicy).To(Equal(corev1.ServiceExternalTrafficPolicyTypeLocal))
		})

		It("should not overwrite .spec.healthCheckNodePort for vpn-shoot service", func() {
			oldVpnSvc := vpnSvc.DeepCopy()
			oldVpnSvc.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeLocal
			oldVpnSvc.Spec.HealthCheckNodePort = 31279

			err := mutator.Mutate(context.TODO(), vpnSvc, oldVpnSvc)

			Expect(err).To(Not(HaveOccurred()))
			Expect(vpnSvc.Spec.ExternalTrafficPolicy).To(Equal(corev1.ServiceExternalTrafficPolicyTypeLocal))
			Expect(vpnSvc.Spec.HealthCheckNodePort).To(Equal(int32(31279)))
		})

		It("should set ExternalTrafficPolicy to Local for addons-nginx-ingress-controller service", func() {
			err := mutator.Mutate(context.TODO(), nginxIngressSvc, nil)

			Expect(err).To(Not(HaveOccurred()))
			Expect(nginxIngressSvc.Spec.ExternalTrafficPolicy).To(Equal(corev1.ServiceExternalTrafficPolicyTypeLocal))
		})

		It("should not overwrite .spec.healthCheckNodePort for addons-nginx-ingress-controller service", func() {
			oldNginxIngressSvc := nginxIngressSvc.DeepCopy()
			oldNginxIngressSvc.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeLocal
			oldNginxIngressSvc.Spec.HealthCheckNodePort = 31280

			err := mutator.Mutate(context.TODO(), nginxIngressSvc, oldNginxIngressSvc)

			Expect(err).To(Not(HaveOccurred()))
			Expect(oldNginxIngressSvc.Spec.ExternalTrafficPolicy).To(Equal(corev1.ServiceExternalTrafficPolicyTypeLocal))
			Expect(oldNginxIngressSvc.Spec.HealthCheckNodePort).To(Equal(int32(31280)))
		})

		It("should not set ExternalTrafficPolicy to Local for other service", func() {
			err := mutator.Mutate(context.TODO(), otherSvc, nil)

			Expect(err).To(Not(HaveOccurred()))
			Expect(otherSvc.Spec.ExternalTrafficPolicy).To(Equal(corev1.ServiceExternalTrafficPolicyTypeCluster))
		})
	})
})
