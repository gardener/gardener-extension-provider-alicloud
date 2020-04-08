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

package shoot

import (
	"context"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Mutator", func() {
	var (
		mutator = NewMutator()
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
