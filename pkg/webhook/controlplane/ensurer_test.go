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

package controlplane

import (
	"context"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/coreos/go-systemd/v22/unit"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/test"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/utils/pointer"
)

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controlplane Webhook Suite")
}

var _ = Describe("Ensurer", func() {
	var (
		ctrl       *gomock.Controller
		eContext20 = gcontext.NewInternalGardenContext(
			&extensionscontroller.Cluster{
				Shoot: &gardencorev1beta1.Shoot{
					Spec: gardencorev1beta1.ShootSpec{
						Kubernetes: gardencorev1beta1.Kubernetes{
							Version: "1.20.0",
						},
					},
				},
			},
		)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#EnsureKubeAPIServerDeployment", func() {
		It("should add missing elements to kube-apiserver deployment", func() {
			var (
				apidep = func() *appsv1.Deployment {
					return &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{Name: v1beta1constants.DeploymentNameKubeAPIServer},
						Spec: appsv1.DeploymentSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "kube-apiserver",
										},
									},
								},
							},
						},
					}
				}
			)

			// Create ensurer
			ensurer := NewEnsurer(logger, false)

			// Call EnsureKubeAPIServerDeployment method and check the result
			dep := apidep()
			err := ensurer.EnsureKubeAPIServerDeployment(context.TODO(), eContext20, dep, nil)
			Expect(err).To(Not(HaveOccurred()))
			checkKubeAPIServerDeployment(dep, []string{})

		})

		It("should modify existing elements of kube-apiserver deployment", func() {
			var (
				apidep = func() *appsv1.Deployment {
					return &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{Name: v1beta1constants.DeploymentNameKubeAPIServer},
						Spec: appsv1.DeploymentSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "kube-apiserver",
											Command: []string{
												"--enable-admission-plugins=Priority,PersistentVolumeLabel",
												"--disable-admission-plugins=",
												"--feature-gates=Foo=true",
											},
										},
									},
								},
							},
						},
					}
				}
			)

			// Create ensurer
			ensurer := NewEnsurer(logger, false)

			// Call EnsureKubeAPIServerDeployment method and check the result
			dep := apidep()
			err := ensurer.EnsureKubeAPIServerDeployment(context.TODO(), eContext20, dep, nil)
			Expect(err).To(Not(HaveOccurred()))
			checkKubeAPIServerDeployment(dep, []string{})
		})
	})

	Describe("#EnsureKubeControllerManagerDeployment", func() {
		It("should add missing elements to kube-controller-manager deployment", func() {
			var (
				dep = &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: v1beta1constants.DeploymentNameKubeControllerManager},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "kube-controller-manager",
									},
								},
							},
						},
					},
				}
			)

			// Create ensurer
			ensurer := NewEnsurer(logger, false)

			// Call EnsureKubeControllerManagerDeployment method and check the result
			err := ensurer.EnsureKubeControllerManagerDeployment(context.TODO(), eContext20, dep, nil)
			Expect(err).To(Not(HaveOccurred()))
			checkKubeControllerManagerDeployment(dep)
		})

		It("should modify existing elements of kube-controller-manager deployment", func() {
			var (
				dep = &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: v1beta1constants.DeploymentNameKubeControllerManager},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "kube-controller-manager",
										Command: []string{
											"--cloud-provider=?",
										},
									},
								},
							},
						},
					},
				}
			)

			// Create ensurer
			ensurer := NewEnsurer(logger, false)

			// Call EnsureKubeControllerManagerDeployment method and check the result
			err := ensurer.EnsureKubeControllerManagerDeployment(context.TODO(), eContext20, dep, nil)
			Expect(err).To(Not(HaveOccurred()))
			checkKubeControllerManagerDeployment(dep)
		})
	})

	Describe("#EnsureKubeletServiceUnitOptions", func() {
		var (
			ensurer        genericmutator.Ensurer
			oldUnitOptions []*unit.UnitOption
		)

		BeforeEach(func() {
			ensurer = NewEnsurer(logger, false)
			oldUnitOptions = []*unit.UnitOption{
				{
					Section: "Service",
					Name:    "ExecStart",
					Value: `/opt/bin/hyperkube kubelet \
    --config=/var/lib/kubelet/config/kubelet`,
				},
			}
		})

		DescribeTable("should modify existing elements of kubelet.service unit options",
			func(gctx gcontext.GardenContext, kubeletVersion *semver.Version, cloudProvider string, equalGreaterV123 bool) {
				valueofPreStart := `/bin/sh -c "echo Z3JlcCAtc3EgUFJPVklERVJfSUQgL3Zhci9saWIva3ViZWxldC9leHRyYV9hcmdzCmlmIFsgJD8gLW5lIDAgXTsgdGhlbgpNRVRBX0VQPWh0dHA6Ly8xMDAuMTAwLjEwMC4yMDAvbGF0ZXN0L21ldGEtZGF0YQpQUk9WSURFUl9JRD1gd2dldCAtcU8tICRNRVRBX0VQL3JlZ2lvbi1pZGAuYHdnZXQgLXFPLSAkTUVUQV9FUC9pbnN0YW5jZS1pZGAKZWNobyBQUk9WSURFUl9JRD0kUFJPVklERVJfSUQgPj4gL3Zhci9saWIva3ViZWxldC9leHRyYV9hcmdzCmVjaG8gUFJPVklERVJfSUQ9JFBST1ZJREVSX0lEIGhhcyBiZWVuIHdyaXR0ZW4gdG8gL3Zhci9saWIva3ViZWxldC9leHRyYV9hcmdzCmZpCg==| base64 -d > /var/lib/kubelet/gardener-set-provider-id && chmod +x /var/lib/kubelet/gardener-set-provider-id && /var/lib/kubelet/gardener-set-provider-id"`
				if equalGreaterV123 {
					valueofPreStart = `/bin/sh -c "echo IyBzZXQgcHJvdmlkZXJpZCBpbiAvdmFyL2xpYi9rdWJlbGV0L2NvbmZpZy9rdWJlbGV0CmdyZXAgLXNxIHBsYWNlX2hvbGRlcl9vZl9wcm92aWRlcmlkIC92YXIvbGliL2t1YmVsZXQvY29uZmlnL2t1YmVsZXQKaWYgWyAkPyAtZXEgMCBdOyB0aGVuCiAgICBNRVRBX0VQPWh0dHA6Ly8xMDAuMTAwLjEwMC4yMDAvbGF0ZXN0L21ldGEtZGF0YQogICAgUFJPVklERVJfSUQ9YHdnZXQgLXFPLSAkTUVUQV9FUC9yZWdpb24taWRgLmB3Z2V0IC1xTy0gJE1FVEFfRVAvaW5zdGFuY2UtaWRgCiAgICBzdWRvIHNlZCAgLWkgInMvcGxhY2VfaG9sZGVyX29mX3Byb3ZpZGVyaWQvJHtQUk9WSURFUl9JRH0vZyIgL3Zhci9saWIva3ViZWxldC9jb25maWcva3ViZWxldAogICAgZWNobyAicHJvdmlkZXJJRD0gJFBST1ZJREVSX0lEIGhhcyBiZWVuIHdyaXR0ZW4gdG8gL3Zhci9saWIva3ViZWxldC9jb25maWcva3ViZWxldCIKZmkK| base64 -d > /var/lib/kubelet/gardener-set-provider-id && chmod +x /var/lib/kubelet/gardener-set-provider-id && /var/lib/kubelet/gardener-set-provider-id"`
				}
				newUnitOptions := []*unit.UnitOption{
					{
						Section: "Service",
						Name:    "ExecStart",
						Value: `/opt/bin/hyperkube kubelet \
    --config=/var/lib/kubelet/config/kubelet`,
					},
					{
						Section: "Service",
						Name:    "ExecStartPre",
						Value:   valueofPreStart,
					},
				}

				if cloudProvider != "" {
					newUnitOptions[0].Value += ` \
    --cloud-provider=` + cloudProvider
				}

				if !equalGreaterV123 {
					newUnitOptions[0].Value += ` \
    --provider-id=${PROVIDER_ID}`
					newUnitOptions[0].Value += ` \
    --enable-controller-attach-detach=true`
				}

				opts, err := ensurer.EnsureKubeletServiceUnitOptions(context.TODO(), gctx, kubeletVersion, oldUnitOptions, nil)
				Expect(err).To(Not(HaveOccurred()))
				Expect(opts).To(Equal(newUnitOptions))
			},

			Entry("kubelet version < 1.23", eContext20, semver.MustParse("1.20.0"), "external", false),
			Entry("kubelet version >= 1.23", eContext20, semver.MustParse("1.23.0"), "external", true),
		)
	})

	Describe("#EnsureKubeletConfiguration", func() {
		var (
			ensurer          genericmutator.Ensurer
			oldKubeletConfig *kubeletconfigv1beta1.KubeletConfiguration
		)

		BeforeEach(func() {
			ensurer = NewEnsurer(logger, false)
			oldKubeletConfig = &kubeletconfigv1beta1.KubeletConfiguration{
				FeatureGates: map[string]bool{
					"Foo": true,
				},
			}
		})

		DescribeTable("should modify existing elements of kubelet configuration",
			func(gctx gcontext.GardenContext, kubeletVersion *semver.Version, csiFeatureGateName string, equalGreaterV123 bool) {
				var (
					enableControllerAttachDetach *bool  = nil
					providerId                   string = ""
				)
				if equalGreaterV123 {
					enableControllerAttachDetach = pointer.Bool(true)
					providerId = "place_holder_of_providerid"
				}
				newKubeletConfig := &kubeletconfigv1beta1.KubeletConfiguration{
					FeatureGates: map[string]bool{
						"Foo": true,
					},
					EnableControllerAttachDetach: enableControllerAttachDetach,
					ProviderID:                   providerId,
				}

				if csiFeatureGateName != "" {
					newKubeletConfig.FeatureGates[csiFeatureGateName] = true
				}

				kubeletConfig := *oldKubeletConfig

				err := ensurer.EnsureKubeletConfiguration(context.TODO(), gctx, kubeletVersion, &kubeletConfig, nil)
				Expect(err).To(Not(HaveOccurred()))
				Expect(&kubeletConfig).To(Equal(newKubeletConfig))
			},

			Entry("kubelet < 1.23", eContext20, semver.MustParse("1.20.0"), "", false),
			Entry("kubelet >= 1.23", eContext20, semver.MustParse("1.23.0"), "", true),
		)
	})
})

func checkKubeAPIServerDeployment(dep *appsv1.Deployment, featureGates []string) {
	// Check that the kube-apiserver container still exists and contains all needed command line args,
	// env vars, and volume mounts
	c := extensionswebhook.ContainerWithName(dep.Spec.Template.Spec.Containers, "kube-apiserver")
	Expect(c).To(Not(BeNil()))
	Expect(c.Command).To(Not(test.ContainElementWithPrefixContaining("--enable-admission-plugins=", "PersistentVolumeLabel", ",")))
	Expect(c.Command).To(test.ContainElementWithPrefixContaining("--disable-admission-plugins=", "PersistentVolumeLabel", ","))
	for _, fg := range featureGates {
		Expect(c.Command).To(test.ContainElementWithPrefixContaining("--feature-gates=", fg, ","))
	}
	Expect(dep.Spec.Template.Labels).To(HaveKeyWithValue("networking.resources.gardener.cloud/to-csi-snapshot-validation-tcp-443", "allowed"))
}

func checkKubeControllerManagerDeployment(dep *appsv1.Deployment) {
	// Check that the kube-controller-manager container still exists and contains all needed command line args,
	// env vars, and volume mounts
	c := extensionswebhook.ContainerWithName(dep.Spec.Template.Spec.Containers, "kube-controller-manager")
	Expect(c).To(Not(BeNil()))
	Expect(c.Command).To(ContainElement("--cloud-provider=external"))
}
