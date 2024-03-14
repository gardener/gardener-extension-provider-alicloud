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
	"encoding/json"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	mockmanager "github.com/gardener/gardener/pkg/mock/controller-runtime/manager"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	fakesecretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager/fake"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/config"
)

const (
	namespace                        = "test"
	genericTokenKubeconfigSecretName = "generic-token-kubeconfig-92e9ae14"
)

var _ = Describe("ValuesProvider", func() {
	var (
		ctrl *gomock.Controller

		// Build scheme
		scheme             = runtime.NewScheme()
		_                  = apisalicloud.AddToScheme(scheme)
		fakeClient         client.Client
		fakeSecretsManager secretsmanager.Interface

		cluster *extensionscontroller.Cluster
		vp      genericactuator.ValuesProvider
		c       *mockclient.MockClient
		mgr     *mockmanager.MockManager

		cp = &extensionsv1alpha1.ControlPlane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "control-plane",
				Namespace: namespace,
			},
			Spec: extensionsv1alpha1.ControlPlaneSpec{
				Region: "eu-central-1",
				SecretRef: corev1.SecretReference{
					Name:      v1beta1constants.SecretNameCloudProvider,
					Namespace: namespace,
				},
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					ProviderConfig: &runtime.RawExtension{
						Raw: encode(&apisalicloud.ControlPlaneConfig{
							CloudControllerManager: &apisalicloud.CloudControllerManagerConfig{
								FeatureGates: map[string]bool{
									"RotateKubeletServerCertificate": true,
								},
							},
							CSI: &apisalicloud.CSI{
								EnableADController: pointer.Bool(true),
							},
						}),
					},
				},
				InfrastructureProviderStatus: &runtime.RawExtension{
					Raw: encode(&apisalicloud.InfrastructureStatus{
						VPC: apisalicloud.VPCStatus{
							ID: "vpc-1234",
							VSwitches: []apisalicloud.VSwitch{
								{
									ID:      "vswitch-acbd1234",
									Purpose: apisalicloud.PurposeNodes,
									Zone:    "eu-central-1a",
								},
							},
						},
					}),
				},
			},
		}

		cidr = "10.250.0.0/19"

		cpSecretKey = client.ObjectKey{Namespace: namespace, Name: v1beta1constants.SecretNameCloudProvider}
		cpSecret    = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      v1beta1constants.SecretNameCloudProvider,
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				alicloud.AccessKeyID:     []byte("foo"),
				alicloud.AccessKeySecret: []byte("bar"),
			},
		}

		checksums = map[string]string{
			v1beta1constants.SecretNameCloudProvider: "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
			alicloud.CloudProviderConfigName:         "08a7bc7fe8f59b055f173145e211760a83f02cf89635cef26ebb351378635606",
		}

		controlPlaneChartValues = map[string]interface{}{
			"global": map[string]interface{}{
				"genericTokenKubeconfigSecretName": genericTokenKubeconfigSecretName,
			},
			"alicloud-cloud-controller-manager": map[string]interface{}{
				"replicas":    1,
				"clusterName": namespace,
				"podNetwork":  cidr,
				"podLabels": map[string]interface{}{
					"maintenance.gardener.cloud/restart": "true",
				},
				"cloudConfig": "{\"Global\":{\"KubernetesClusterTag\":\"test\",\"clusterID\":\"test\",\"uid\":\"\",\"vpcid\":\"vpc-1234\",\"region\":\"eu-central-1\",\"zoneid\":\"eu-central-1a\",\"vswitchid\":\"vswitch-acbd1234\",\"accessKeyID\":\"Zm9v\",\"accessKeySecret\":\"YmFy\"}}",
				"featureGates": map[string]bool{
					"RotateKubeletServerCertificate": true,
				},
				"ccmNetworkFalg": "public",
			},
			"csi-alicloud": map[string]interface{}{
				"replicas":           1,
				"regionID":           "eu-central-1",
				"enableADController": true,
				"csiPluginController": map[string]interface{}{
					"snapshotPrefix":         "myshoot",
					"persistentVolumePrefix": "myshoot",
					"podAnnotations": map[string]interface{}{
						"checksum/secret-cloudprovider": "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
					},
				},

				"csiSnapshotController": map[string]interface{}{},
				"csiSnapshotValidationWebhook": map[string]interface{}{
					"replicas": 1,
					"secrets": map[string]interface{}{
						"server": "csi-snapshot-validation-server",
					},
					"topologyAwareRoutingEnabled": false,
				},
			},
		}

		controlPlaneShootChartValues = map[string]interface{}{
			"csi-alicloud": map[string]interface{}{
				"credential": map[string]interface{}{
					"accessKeyID":     "Zm9v",
					"accessKeySecret": "YmFy",
				},
				"enableADController": true,
				"vpaEnabled":         true,
				"webhookConfig": map[string]interface{}{
					"url":      "https://" + alicloud.CSISnapshotValidationName + "." + cp.Namespace + "/volumesnapshot",
					"caBundle": "",
				},
				"pspDisabled": false,
			},
		}

		csi = config.CSI{}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		fakeClient = fakeclient.NewClientBuilder().Build()
		fakeSecretsManager = fakesecretsmanager.New(fakeClient, namespace)

		cluster = &extensionscontroller.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"generic-token-kubeconfig.secret.gardener.cloud/name": genericTokenKubeconfigSecretName,
				},
			},
			Seed: &gardencorev1beta1.Seed{},
			Shoot: &gardencorev1beta1.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myshoot",
				},
				Spec: gardencorev1beta1.ShootSpec{
					Provider: gardencorev1beta1.Provider{
						Workers: []gardencorev1beta1.Worker{
							{
								Name: "worker",
							},
						},
					},
					Networking: &gardencorev1beta1.Networking{
						Pods: &cidr,
					},
					Kubernetes: gardencorev1beta1.Kubernetes{
						Version: "1.24.0",
						VerticalPodAutoscaler: &gardencorev1beta1.VerticalPodAutoscaler{
							Enabled: true,
						},
					},
				},
			},
		}

		c = mockclient.NewMockClient(ctrl)
		mgr = mockmanager.NewMockManager(ctrl)
		mgr.EXPECT().GetClient().Return(c)
		mgr.EXPECT().GetScheme().Return(scheme).Times(2)
		vp = NewValuesProvider(mgr, csi)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#GetControlPlaneChartValues", func() {
		BeforeEach(func() {
			c.EXPECT().Get(context.TODO(), cpSecretKey, &corev1.Secret{}).DoAndReturn(clientGet(cpSecret))

			By("creating secrets managed outside of this package for whose secretsmanager.Get() will be called")
			Expect(fakeClient.Create(context.TODO(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ca-provider-alicloud-controlplane", Namespace: namespace}})).To(Succeed())
			Expect(fakeClient.Create(context.TODO(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "csi-snapshot-validation-server", Namespace: namespace}})).To(Succeed())

			c.EXPECT().Delete(context.TODO(), &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "allow-kube-apiserver-to-csi-snapshot-validation", Namespace: cp.Namespace}})
		})

		It("should return correct control plane chart values", func() {
			// Call GetControlPlaneChartValues method and check the result
			values, err := vp.GetControlPlaneChartValues(context.TODO(), cp, cluster, fakeSecretsManager, checksums, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(controlPlaneChartValues))
		})

		It("should set chart values ccmNetworkFalg vpc when seed provider type is alicloud", func() {
			// Call GetControlPlaneChartValues method and check the result
			cluster.Seed = &gardencorev1beta1.Seed{
				Spec: gardencorev1beta1.SeedSpec{
					Provider: gardencorev1beta1.SeedProvider{
						Type: "alicloud",
						Region: "region",
					},
				},
			}
			cluster.Shoot = &gardencorev1beta1.Shoot{
				Spec: gardencorev1beta1.ShootSpec{
					Region: "region",
				},
			}
			values, err := vp.GetControlPlaneChartValues(context.TODO(), cp, cluster, fakeSecretsManager, checksums, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(HaveKey("alicloud-cloud-controller-manager"))
			Expect(values["alicloud-cloud-controller-manager"]).To(HaveKeyWithValue("ccmNetworkFalg", "vpc"))
		})

		DescribeTable("topologyAwareRoutingEnabled value",
			func(seedSettings *gardencorev1beta1.SeedSettings, shootControlPlane *gardencorev1beta1.ControlPlane, expected bool) {
				cluster.Seed = &gardencorev1beta1.Seed{
					Spec: gardencorev1beta1.SeedSpec{
						Settings: seedSettings,
					},
				}
				cluster.Shoot.Spec.ControlPlane = shootControlPlane

				values, err := vp.GetControlPlaneChartValues(context.TODO(), cp, cluster, fakeSecretsManager, checksums, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(values).To(HaveKey("csi-alicloud"))
				Expect(values["csi-alicloud"]).To(HaveKeyWithValue("csiSnapshotValidationWebhook", HaveKeyWithValue("topologyAwareRoutingEnabled", expected)))
			},

			Entry("seed setting is nil, shoot control plane is not HA",
				nil,
				&gardencorev1beta1.ControlPlane{HighAvailability: nil},
				false,
			),
			Entry("seed setting is disabled, shoot control plane is not HA",
				&gardencorev1beta1.SeedSettings{TopologyAwareRouting: &gardencorev1beta1.SeedSettingTopologyAwareRouting{Enabled: false}},
				&gardencorev1beta1.ControlPlane{HighAvailability: nil},
				false,
			),
			Entry("seed setting is enabled, shoot control plane is not HA",
				&gardencorev1beta1.SeedSettings{TopologyAwareRouting: &gardencorev1beta1.SeedSettingTopologyAwareRouting{Enabled: true}},
				&gardencorev1beta1.ControlPlane{HighAvailability: nil},
				false,
			),
			Entry("seed setting is nil, shoot control plane is HA with failure tolerance type 'zone'",
				nil,
				&gardencorev1beta1.ControlPlane{HighAvailability: &gardencorev1beta1.HighAvailability{FailureTolerance: gardencorev1beta1.FailureTolerance{Type: gardencorev1beta1.FailureToleranceTypeZone}}},
				false,
			),
			Entry("seed setting is disabled, shoot control plane is HA with failure tolerance type 'zone'",
				&gardencorev1beta1.SeedSettings{TopologyAwareRouting: &gardencorev1beta1.SeedSettingTopologyAwareRouting{Enabled: false}},
				&gardencorev1beta1.ControlPlane{HighAvailability: &gardencorev1beta1.HighAvailability{FailureTolerance: gardencorev1beta1.FailureTolerance{Type: gardencorev1beta1.FailureToleranceTypeZone}}},
				false,
			),
			Entry("seed setting is enabled, shoot control plane is HA with failure tolerance type 'zone'",
				&gardencorev1beta1.SeedSettings{TopologyAwareRouting: &gardencorev1beta1.SeedSettingTopologyAwareRouting{Enabled: true}},
				&gardencorev1beta1.ControlPlane{HighAvailability: &gardencorev1beta1.HighAvailability{FailureTolerance: gardencorev1beta1.FailureTolerance{Type: gardencorev1beta1.FailureToleranceTypeZone}}},
				true,
			),
		)
	})

	Describe("#GetControlPlaneShootChartValues", func() {
		BeforeEach(func() {
			c.EXPECT().Get(context.TODO(), cpSecretKey, &corev1.Secret{}).DoAndReturn(clientGet(cpSecret))

			By("creating secrets managed outside of this package for whose secretsmanager.Get() will be called")
			Expect(fakeClient.Create(context.TODO(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ca-provider-alicloud-controlplane", Namespace: namespace}})).To(Succeed())
			Expect(fakeClient.Create(context.TODO(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "csi-snapshot-validation-server", Namespace: namespace}})).To(Succeed())
		})

		It("should return correct control plane shoot chart values", func() {
			// Call GetControlPlaneShootChartValues method and check the result
			values, err := vp.GetControlPlaneShootChartValues(context.TODO(), cp, cluster, fakeSecretsManager, checksums)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(controlPlaneShootChartValues))
		})

		Context("podSecurityPolicy", func() {
			It("should return correct shoot control plane chart when PodSecurityPolicy admission plugin is not disabled in the shoot", func() {
				cluster.Shoot.Spec.Kubernetes.KubeAPIServer = &gardencorev1beta1.KubeAPIServerConfig{
					AdmissionPlugins: []gardencorev1beta1.AdmissionPlugin{
						{
							Name: "PodSecurityPolicy",
						},
					},
				}

				// Call GetControlPlaneShootChartValues method and check the result
				values, err := vp.GetControlPlaneShootChartValues(context.TODO(), cp, cluster, fakeSecretsManager, checksums)
				Expect(err).NotTo(HaveOccurred())

				controlPlaneShootChartValues := map[string]interface{}{
					"csi-alicloud": map[string]interface{}{
						"credential": map[string]interface{}{
							"accessKeyID":     "Zm9v",
							"accessKeySecret": "YmFy",
						},
						"enableADController": true,
						"vpaEnabled":         true,
						"webhookConfig": map[string]interface{}{
							"url":      "https://" + alicloud.CSISnapshotValidationName + "." + cp.Namespace + "/volumesnapshot",
							"caBundle": "",
						},
						"pspDisabled": false,
					},
				}

				Expect(values).To(Equal(controlPlaneShootChartValues))
			})

			It("should return correct shoot control plane chart when PodSecurityPolicy admission plugin is disabled in the shoot", func() {
				cluster.Shoot.Spec.Kubernetes.KubeAPIServer = &gardencorev1beta1.KubeAPIServerConfig{
					AdmissionPlugins: []gardencorev1beta1.AdmissionPlugin{
						{
							Name:     "PodSecurityPolicy",
							Disabled: pointer.Bool(true),
						},
					},
				}

				// Call GetControlPlaneShootChartValues method and check the result
				values, err := vp.GetControlPlaneShootChartValues(context.TODO(), cp, cluster, fakeSecretsManager, checksums)
				Expect(err).NotTo(HaveOccurred())

				controlPlaneShootChartValues := map[string]interface{}{
					"csi-alicloud": map[string]interface{}{
						"credential": map[string]interface{}{
							"accessKeyID":     "Zm9v",
							"accessKeySecret": "YmFy",
						},
						"enableADController": true,
						"vpaEnabled":         true,
						"webhookConfig": map[string]interface{}{
							"url":      "https://" + alicloud.CSISnapshotValidationName + "." + cp.Namespace + "/volumesnapshot",
							"caBundle": "",
						},
						"pspDisabled": true,
					},
				}
				Expect(values).To(Equal(controlPlaneShootChartValues))
			})
		})
	})
})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}

func clientGet(result runtime.Object) interface{} {
	return func(ctx context.Context, key client.ObjectKey, obj runtime.Object, _ ...client.GetOption) error {
		switch obj.(type) {
		case *corev1.Secret:
			*obj.(*corev1.Secret) = *result.(*corev1.Secret)
		}
		return nil
	}
}
