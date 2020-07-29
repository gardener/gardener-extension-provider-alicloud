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
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

const (
	namespace = "test"
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
			Shoot: &gardencorev1beta1.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myshoot",
				},
				Spec: gardencorev1beta1.ShootSpec{
					Networking: gardencorev1beta1.Networking{
						Pods: &cidr,
					},
					Kubernetes: gardencorev1beta1.Kubernetes{
						Version: "1.14.0",
						VerticalPodAutoscaler: &gardencorev1beta1.VerticalPodAutoscaler{
							Enabled: true,
						},
					},
				},
			},
		}

		// TODO remove cpMap in next version
		cpConfigmap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "cloud-provider-config",
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
			"cloud-provider-config":                  "08a7bc7fe8f59b055f173145e211760a83f02cf89635cef26ebb351378635606",
			"cloud-controller-manager":               "3d791b164a808638da9a8df03924be2a41e34cd664e42231c00fe369e3588272",
			"csi-attacher":                           "2da58ad61c401a2af779a909d22fb42eed93a1524cbfdab974ceedb413fcb914",
			"csi-provisioner":                        "f75b42d40ab501428c383dfb2336cb1fc892bbee1fc1d739675171e4acc4d911",
			"csi-snapshotter":                        "bf417dd97dc3e8c2092bb5b2ba7b0f1093ebc4bb5952091ee554cf5b7ea74508",
			"csi-snapshot-controller":                "4b7977b627acb0c1f74607cb31bd3a706a18ff1067121e31cec5e432a8f6ab15",
			"csi-resizer":                            "5df115bd53f09da2d6d27bfb048c14dabd14a66608cfdba5ecd2d0687889cc6a",
		}

		controlPlaneChartValues = map[string]interface{}{
			"alicloud-cloud-controller-manager": map[string]interface{}{
				"replicas":          1,
				"clusterName":       namespace,
				"kubernetesVersion": "1.14.0",
				"podNetwork":        cidr,
				"podAnnotations": map[string]interface{}{
					"checksum/secret-cloud-controller-manager": "3d791b164a808638da9a8df03924be2a41e34cd664e42231c00fe369e3588272",
					"checksum/secret-cloud-provider-config":    "08a7bc7fe8f59b055f173145e211760a83f02cf89635cef26ebb351378635606",
				},
				"podLabels": map[string]interface{}{
					"maintenance.gardener.cloud/restart": "true",
				},
				"cloudConfig": "{\"Global\":{\"KubernetesClusterTag\":\"test\",\"clusterID\":\"test\",\"uid\":\"\",\"vpcid\":\"vpc-1234\",\"region\":\"eu-central-1\",\"zoneid\":\"eu-central-1a\",\"vswitchid\":\"vswitch-acbd1234\",\"accessKeyID\":\"Zm9v\",\"accessKeySecret\":\"YmFy\"}}",
				"featureGates": map[string]bool{
					"CustomResourceValidation": true,
				},
			},
			"csi-alicloud": map[string]interface{}{
				"replicas":          1,
				"regionID":          "eu-central-1",
				"kubernetesVersion": "1.14.0",

				"csiPluginController": map[string]interface{}{
					"snapshotPrefix":         "myshoot",
					"persistentVolumePrefix": "myshoot",
					"podAnnotations": map[string]interface{}{
						"checksum/secret-csi-attacher":    "2da58ad61c401a2af779a909d22fb42eed93a1524cbfdab974ceedb413fcb914",
						"checksum/secret-csi-provisioner": "f75b42d40ab501428c383dfb2336cb1fc892bbee1fc1d739675171e4acc4d911",
						"checksum/secret-csi-snapshotter": "bf417dd97dc3e8c2092bb5b2ba7b0f1093ebc4bb5952091ee554cf5b7ea74508",
						"checksum/secret-csi-resizer":     "5df115bd53f09da2d6d27bfb048c14dabd14a66608cfdba5ecd2d0687889cc6a",
						"checksum/secret-cloudprovider":   "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
					},
				},

				"csiSnapshotController": map[string]interface{}{
					"podAnnotations": map[string]interface{}{
						"checksum/secret-csi-snapshot-controller": "4b7977b627acb0c1f74607cb31bd3a706a18ff1067121e31cec5e432a8f6ab15",
					},
				},
			},
		}

		controlPlaneShootChartValues = map[string]interface{}{
			"csi-alicloud": map[string]interface{}{
				"credential": map[string]interface{}{
					"accessKeyID":     "Zm9v",
					"accessKeySecret": "YmFy",
				},
				"kubernetesVersion": "1.14.0",
				"vpaEnabled":        true,
			},
		}

		logger = log.Log.WithName("test")
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
			client.EXPECT().Delete(context.TODO(), cpConfigmap).DoAndReturn(clientDeleteSuccess())
			// Create valuesProvider
			vp := NewValuesProvider(logger)
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
			vp := NewValuesProvider(logger)
			err := vp.(inject.Client).InjectClient(client)
			Expect(err).NotTo(HaveOccurred())

			// Call GetControlPlaneShootChartValues method and check the result
			values, err := vp.GetControlPlaneShootChartValues(context.TODO(), cp, cluster, checksums)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(controlPlaneShootChartValues))
		})
	})
})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}

func clientDeleteSuccess() interface{} {
	return func(ctx context.Context, cm runtime.Object) error {
		return nil
	}
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
