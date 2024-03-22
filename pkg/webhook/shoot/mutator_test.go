// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot

import (
	"context"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/config"
)

var _ = Describe("Mutator", func() {
	var (
		service = &config.Service{BackendLoadBalancerSpec: "slb.s1.small"}

		mutator = NewMutator(service)
		dep     = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "metrics-server"},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "metrics-server",
								Command: []string{
									"--profiling=false",
									"--cert-dir=/home/certdir",
									"--secure-port=8443",
									"--kubelet-insecure-tls",
									"--tls-cert-file=/srv/metrics-server/tls/tls.crt",
									"--tls-private-key-file=/srv/metrics-server/tls/tls.key",
									"--v=2",
								},
							},
						},
					},
				},
			},
		}
	)
	Describe("#MutateMetricsServerDeployment", func() {
		It("should modify existing elements of metrics-server deployment", func() {
			err := mutator.Mutate(context.TODO(), dep, nil)
			c := extensionswebhook.ContainerWithName(dep.Spec.Template.Spec.Containers, "metrics-server")
			Expect(c).To(Not(BeNil()))
			Expect(c.Command).To(ContainElement("--kubelet-preferred-address-types=InternalIP"))
			Expect(err).To(Not(HaveOccurred()))
		})
	})
})
