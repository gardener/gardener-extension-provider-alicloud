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

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/config"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

const (
	namespace                        = "test"
	genericTokenKubeconfigSecretName = "generic-token-kubeconfig-92e9ae14"
)

var _ = Describe("ValuesProvider", func() {
	var (
		ctrl *gomock.Controller

		// Build scheme
		scheme = runtime.NewScheme()
		_      = apisalicloud.AddToScheme(scheme)

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
									"CustomResourceValidation": true,
								},
							},
							CSI: &apisalicloud.CSI{
								EnableADController: pointer.BoolPtr(true),
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

		cidr    = "10.250.0.0/19"
		cluster = &extensionscontroller.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"generic-token-kubeconfig.secret.gardener.cloud/name": genericTokenKubeconfigSecretName,
				},
			},
			Shoot: &gardencorev1beta1.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myshoot",
				},
				Spec: gardencorev1beta1.ShootSpec{
					Networking: gardencorev1beta1.Networking{
						Pods: &cidr,
					},
					Kubernetes: gardencorev1beta1.Kubernetes{
						Version: "1.20.0",
						VerticalPodAutoscaler: &gardencorev1beta1.VerticalPodAutoscaler{
							Enabled: true,
						},
					},
				},
			},
		}

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
		}

		controlPlaneChartValues = map[string]interface{}{
			"global": map[string]interface{}{
				"genericTokenKubeconfigSecretName": genericTokenKubeconfigSecretName,
			},
			"alicloud-cloud-controller-manager": map[string]interface{}{
				"replicas":          1,
				"clusterName":       namespace,
				"kubernetesVersion": "1.20.0",
				"podNetwork":        cidr,
				"podLabels": map[string]interface{}{
					"maintenance.gardener.cloud/restart": "true",
				},
				"cloudConfig": "{\"Global\":{\"KubernetesClusterTag\":\"test\",\"clusterID\":\"test\",\"uid\":\"\",\"vpcid\":\"vpc-1234\",\"region\":\"eu-central-1\",\"zoneid\":\"eu-central-1a\",\"vswitchid\":\"vswitch-acbd1234\",\"accessKeyID\":\"Zm9v\",\"accessKeySecret\":\"YmFy\"}}",
				"featureGates": map[string]bool{
					"CustomResourceValidation": true,
				},
			},
			"csi-alicloud": map[string]interface{}{
				"replicas":           1,
				"regionID":           "eu-central-1",
				"kubernetesVersion":  "1.20.0",
				"enableADController": true,
				"csiPluginController": map[string]interface{}{
					"snapshotPrefix":         "myshoot",
					"persistentVolumePrefix": "myshoot",
					"podAnnotations": map[string]interface{}{
						"checksum/secret-cloudprovider": "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
					},
				},

				"csiSnapshotController": map[string]interface{}{},
			},
		}

		controlPlaneShootChartValues = map[string]interface{}{
			"csi-alicloud": map[string]interface{}{
				"credential": map[string]interface{}{
					"accessKeyID":     "Zm9v",
					"accessKeySecret": "YmFy",
				},
				"kubernetesVersion":  "1.20.0",
				"enableADController": true,
				"vpaEnabled":         true,
			},
		}

		logger = log.Log.WithName("test")
		csi    = config.CSI{}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#GetControlPlaneChartValues", func() {
		It("should return correct control plane chart values", func() {
			// Create mock client
			client := mockclient.NewMockClient(ctrl)
			client.EXPECT().Get(context.TODO(), cpSecretKey, &corev1.Secret{}).DoAndReturn(clientGet(cpSecret))
			// Create valuesProvider
			vp := NewValuesProvider(logger, csi)
			err := vp.(inject.Scheme).InjectScheme(scheme)
			Expect(err).NotTo(HaveOccurred())
			err = vp.(inject.Client).InjectClient(client)
			Expect(err).NotTo(HaveOccurred())

			// Call GetControlPlaneChartValues method and check the result
			values, err := vp.GetControlPlaneChartValues(context.TODO(), cp, cluster, checksums, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(controlPlaneChartValues))
		})
	})

	Describe("#GetControlPlaneShootChartValues", func() {
		It("should return correct control plane shoot chart values", func() {
			// Create mock client
			client := mockclient.NewMockClient(ctrl)
			client.EXPECT().Get(context.TODO(), cpSecretKey, &corev1.Secret{}).DoAndReturn(clientGet(cpSecret))

			// Create valuesProvider
			vp := NewValuesProvider(logger, csi)
			err := vp.(inject.Scheme).InjectScheme(scheme)
			Expect(err).NotTo(HaveOccurred())
			err = vp.(inject.Client).InjectClient(client)
			Expect(err).NotTo(HaveOccurred())

			// Call GetControlPlaneShootChartValues method and check the result
			values, err := vp.GetControlPlaneShootChartValues(context.TODO(), cp, cluster, checksums)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(controlPlaneShootChartValues))
		})
	})

	Describe("#GetControlPlaneShootCRDsChartValues", func() {
		It("should return correct control plane shoot CRDs chart values ", func() {
			vp := NewValuesProvider(logger, csi)

			values, err := vp.GetControlPlaneShootCRDsChartValues(context.TODO(), cp, cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(map[string]interface{}{"volumesnapshots": map[string]interface{}{"kubernetesVersion": "1.20.0"}}))
		})
	})
})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}

func clientGet(result runtime.Object) interface{} {
	return func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
		switch obj.(type) {
		case *corev1.Secret:
			*obj.(*corev1.Secret) = *result.(*corev1.Secret)
		}
		return nil
	}
}
