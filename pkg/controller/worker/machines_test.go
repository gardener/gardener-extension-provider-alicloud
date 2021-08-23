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

package worker_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"k8s.io/utils/pointer"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	apiv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/worker"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	genericworkeractuator "github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	mockkubernetes "github.com/gardener/gardener/pkg/client/kubernetes/mock"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Machines", func() {
	var (
		ctrl         *gomock.Controller
		c            *mockclient.MockClient
		statusWriter *mockclient.MockStatusWriter
		chartApplier *mockkubernetes.MockChartApplier
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		c = mockclient.NewMockClient(ctrl)
		statusWriter = mockclient.NewMockStatusWriter(ctrl)
		chartApplier = mockkubernetes.NewMockChartApplier(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("workerDelegate", func() {
		workerDelegate, _ := NewWorkerDelegate(common.NewClientContext(nil, nil, nil), nil, "", nil, nil)

		Describe("#MachineClassKind", func() {
			It("should return the correct kind of the machine class", func() {
				Expect(workerDelegate.MachineClassKind()).To(Equal("MachineClass"))
			})
		})

		Describe("#MachineClass", func() {
			It("should return the correct type for the machine class", func() {
				Expect(workerDelegate.MachineClass()).To(Equal(&machinev1alpha1.MachineClass{}))
			})
		})

		Describe("#MachineClassList", func() {
			It("should return the correct type for the machine class list", func() {
				Expect(workerDelegate.MachineClassList()).To(Equal(&machinev1alpha1.MachineClassList{}))
			})
		})

		Describe("#GenerateMachineDeployments, #DeployMachineClasses", func() {
			var (
				namespace        string
				cloudProfileName string

				region string

				machineImageName        string
				machineImageVersion     string
				machineImageID          string
				encryptedImageID        string
				instanceChargeType      string
				internetChargeType      string
				internetMaxBandwidthIn  int
				internetMaxBandwidthOut int
				spotStrategy            string

				machineType     string
				userData        []byte
				securityGroupID string

				volumeType           string
				volumeSize           int
				volume2Encrypted     bool
				dataVolume1Name      string
				dataVolume1Size      int
				dataVolume1Type      string
				dataVolume1Encrypted bool
				dataVolume2Name      string
				dataVolume2Size      int
				dataVolume2Type      string
				dataVolume2Encrypted bool

				namePool1           string
				minPool1            int32
				maxPool1            int32
				maxSurgePool1       intstr.IntOrString
				maxUnavailablePool1 intstr.IntOrString

				namePool2           string
				minPool2            int32
				maxPool2            int32
				maxSurgePool2       intstr.IntOrString
				maxUnavailablePool2 intstr.IntOrString

				vswitchZone1 string
				vswitchZone2 string
				zone1        string
				zone2        string

				machineConfiguration *machinev1alpha1.MachineConfiguration

				workerPoolHash1 string
				workerPoolHash2 string

				shootVersionMajorMinor string
				shootVersion           string
				scheme                 *runtime.Scheme
				decoder                runtime.Decoder
				clusterWithoutImages   *extensionscontroller.Cluster
				cluster                *extensionscontroller.Cluster
				w                      *extensionsv1alpha1.Worker
			)

			BeforeEach(func() {
				namespace = "shoot--foobar--alicloud"
				cloudProfileName = "alicloud"

				region = "china"

				machineImageName = "my-os"
				machineImageVersion = "123"
				machineImageID = "ami-123456"
				encryptedImageID = "ami-123456-encrypted"
				instanceChargeType = "PostPaid"
				internetChargeType = "PayByTraffic"
				internetMaxBandwidthIn = 5
				internetMaxBandwidthOut = 5
				spotStrategy = "NoSpot"

				machineType = "large"
				userData = []byte("some-user-data")
				securityGroupID = "sg-12345"

				volumeType = "normal"
				volumeSize = 20
				volume2Encrypted = true
				dataVolume1Name = "d1"
				dataVolume1Size = 21
				dataVolume1Type = "special"
				dataVolume1Encrypted = false
				dataVolume2Name = "d2"
				dataVolume2Size = 22
				dataVolume2Type = "superspecial"
				dataVolume2Encrypted = true

				namePool1 = "pool-1"
				minPool1 = 5
				maxPool1 = 10
				maxSurgePool1 = intstr.FromInt(3)
				maxUnavailablePool1 = intstr.FromInt(2)

				namePool2 = "pool-2"
				minPool2 = 30
				maxPool2 = 45
				maxSurgePool2 = intstr.FromInt(10)
				maxUnavailablePool2 = intstr.FromInt(15)

				vswitchZone1 = "vswitch-acbd1234"
				vswitchZone2 = "vswitch-4321dbca"
				zone1 = region + "a"
				zone2 = region + "b"

				machineConfiguration = &machinev1alpha1.MachineConfiguration{}

				shootVersionMajorMinor = "1.2"
				shootVersion = shootVersionMajorMinor + ".3"

				clusterWithoutImages = &extensionscontroller.Cluster{
					Shoot: &gardencorev1beta1.Shoot{
						Spec: gardencorev1beta1.ShootSpec{
							Kubernetes: gardencorev1beta1.Kubernetes{
								Version: shootVersion,
							},
						},
					},
				}

				cloudProfileConfig := &apiv1alpha1.CloudProfileConfig{
					TypeMeta: metav1.TypeMeta{
						APIVersion: apiv1alpha1.SchemeGroupVersion.String(),
						Kind:       "CloudProfileConfig",
					},
					MachineImages: []apiv1alpha1.MachineImages{
						{
							Name: machineImageName,
							Versions: []apiv1alpha1.MachineImageVersion{
								{
									Version: machineImageVersion,
									Regions: []apiv1alpha1.RegionIDMapping{
										{
											Name: region,
											ID:   machineImageID,
										},
									},
								},
							},
						},
					},
				}
				cloudProfileConfigJSON, _ := json.Marshal(cloudProfileConfig)
				cluster = &extensionscontroller.Cluster{
					CloudProfile: &gardencorev1beta1.CloudProfile{
						ObjectMeta: metav1.ObjectMeta{
							Name: cloudProfileName,
						},
						Spec: gardencorev1beta1.CloudProfileSpec{
							ProviderConfig: &runtime.RawExtension{
								Raw: cloudProfileConfigJSON,
							},
						},
					},
					Shoot: clusterWithoutImages.Shoot,
				}

				w = &extensionsv1alpha1.Worker{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
					},
					Spec: extensionsv1alpha1.WorkerSpec{
						SecretRef: corev1.SecretReference{
							Name:      "secret",
							Namespace: namespace,
						},
						Region: region,
						InfrastructureProviderStatus: &runtime.RawExtension{
							Raw: encode(&api.InfrastructureStatus{
								VPC: api.VPCStatus{
									VSwitches: []api.VSwitch{
										{
											ID:      vswitchZone1,
											Purpose: "nodes",
											Zone:    zone1,
										},
										{
											ID:      vswitchZone2,
											Purpose: "nodes",
											Zone:    zone2,
										},
									},
									SecurityGroups: []api.SecurityGroup{
										{
											ID:      securityGroupID,
											Purpose: "nodes",
										},
									},
								},
								MachineImages: []api.MachineImage{
									{
										Name:      machineImageName,
										Version:   machineImageVersion,
										Encrypted: pointer.BoolPtr(true),
										ID:        encryptedImageID,
									},
								},
							}),
						},
						Pools: []extensionsv1alpha1.WorkerPool{
							{
								Name:           namePool1,
								Minimum:        minPool1,
								Maximum:        maxPool1,
								MaxSurge:       maxSurgePool1,
								MaxUnavailable: maxUnavailablePool1,
								MachineType:    machineType,
								MachineImage: extensionsv1alpha1.MachineImage{
									Name:    machineImageName,
									Version: machineImageVersion,
								},
								UserData: userData,
								Volume: &extensionsv1alpha1.Volume{
									Type: &volumeType,
									Size: fmt.Sprintf("%dGi", volumeSize),
								},
								DataVolumes: []extensionsv1alpha1.DataVolume{
									{
										Name:      dataVolume1Name,
										Size:      fmt.Sprintf("%dGi", dataVolume1Size),
										Type:      &dataVolume1Type,
										Encrypted: &dataVolume1Encrypted,
									},
									{
										Name:      dataVolume2Name,
										Size:      fmt.Sprintf("%dGi", dataVolume2Size),
										Type:      &dataVolume2Type,
										Encrypted: &dataVolume2Encrypted,
									},
								},
								Zones: []string{
									zone1,
									zone2,
								},
							},
							{
								Name:           namePool2,
								Minimum:        minPool2,
								Maximum:        maxPool2,
								MaxSurge:       maxSurgePool2,
								MaxUnavailable: maxUnavailablePool2,
								MachineType:    machineType,
								MachineImage: extensionsv1alpha1.MachineImage{
									Name:    machineImageName,
									Version: machineImageVersion,
								},
								UserData: userData,
								Volume: &extensionsv1alpha1.Volume{
									Type:      &volumeType,
									Size:      fmt.Sprintf("%dGi", volumeSize),
									Encrypted: &volume2Encrypted,
								},
								Zones: []string{
									zone1,
									zone2,
								},
							},
						},
					},
				}

				scheme = runtime.NewScheme()
				_ = api.AddToScheme(scheme)
				_ = apiv1alpha1.AddToScheme(scheme)
				decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()

				workerPoolHash1, _ = worker.WorkerPoolHash(w.Spec.Pools[0], cluster, fmt.Sprintf("%dGi", dataVolume1Size), dataVolume1Type, strconv.FormatBool(dataVolume1Encrypted), fmt.Sprintf("%dGi", dataVolume2Size), dataVolume2Type, strconv.FormatBool(dataVolume2Encrypted))
				workerPoolHash2, _ = worker.WorkerPoolHash(w.Spec.Pools[1], cluster, "true")

				workerDelegate, _ = NewWorkerDelegate(common.NewClientContext(c, scheme, decoder), chartApplier, "", w, clusterWithoutImages)
			})

			Describe("machine images", func() {
				var (
					defaultMachineClass map[string]interface{}
					machineDeployments  worker.MachineDeployments
					machineClasses      map[string]interface{}
				)

				BeforeEach(func() {
					defaultMachineClass = map[string]interface{}{
						"imageID":         machineImageID,
						"instanceType":    machineType,
						"region":          region,
						"securityGroupID": securityGroupID,
						"systemDisk": map[string]interface{}{
							"category": volumeType,
							"size":     volumeSize,
						},
						"instanceChargeType":      instanceChargeType,
						"internetChargeType":      internetChargeType,
						"internetMaxBandwidthIn":  internetMaxBandwidthIn,
						"internetMaxBandwidthOut": internetMaxBandwidthOut,
						"spotStrategy":            spotStrategy,
						"tags": map[string]string{
							fmt.Sprintf("kubernetes.io/cluster/%s", namespace):     "1",
							fmt.Sprintf("kubernetes.io/role/worker/%s", namespace): "1",
						},
						"secret": map[string]interface{}{
							"userData": string(userData),
						},
						"credentialsSecretRef": map[string]interface{}{
							"name":      w.Spec.SecretRef.Name,
							"namespace": w.Spec.SecretRef.Namespace,
						},
					}

					dataDisksPool1 := []map[string]interface{}{
						{
							"name":               dataVolume1Name,
							"size":               dataVolume1Size,
							"deleteWithInstance": true,
							"category":           dataVolume1Type,
							"encrypted":          dataVolume1Encrypted,
							"description":        namespace + "-datavol-" + dataVolume1Name,
						},
						{
							"name":               dataVolume2Name,
							"size":               dataVolume2Size,
							"deleteWithInstance": true,
							"category":           dataVolume2Type,
							"encrypted":          dataVolume2Encrypted,
							"description":        namespace + "-datavol-" + dataVolume2Name,
						},
					}

					var (
						machineClassPool1Zone1 = useDefaultMachineClass(defaultMachineClass,
							"vSwitchID", vswitchZone1,
							"zoneID", zone1,
							"dataDisks", dataDisksPool1,
						)
						machineClassPool1Zone2 = useDefaultMachineClass(defaultMachineClass,
							"vSwitchID", vswitchZone2,
							"zoneID", zone2,
							"dataDisks", dataDisksPool1,
						)
						machineClassPool2Zone1 = useDefaultMachineClass(defaultMachineClass,
							"vSwitchID", vswitchZone1,
							"zoneID", zone1,
							"imageID", encryptedImageID,
						)
						machineClassPool2Zone2 = useDefaultMachineClass(defaultMachineClass,
							"vSwitchID", vswitchZone2,
							"zoneID", zone2,
							"imageID", encryptedImageID,
						)

						machineClassNamePool1Zone1 = fmt.Sprintf("%s-%s-%s", namespace, namePool1, zone1)
						machineClassNamePool1Zone2 = fmt.Sprintf("%s-%s-%s", namespace, namePool1, zone2)
						machineClassNamePool2Zone1 = fmt.Sprintf("%s-%s-%s", namespace, namePool2, zone1)
						machineClassNamePool2Zone2 = fmt.Sprintf("%s-%s-%s", namespace, namePool2, zone2)

						machineClassWithHashPool1Zone1 = fmt.Sprintf("%s-%s", machineClassNamePool1Zone1, workerPoolHash1)
						machineClassWithHashPool1Zone2 = fmt.Sprintf("%s-%s", machineClassNamePool1Zone2, workerPoolHash1)
						machineClassWithHashPool2Zone1 = fmt.Sprintf("%s-%s", machineClassNamePool2Zone1, workerPoolHash2)
						machineClassWithHashPool2Zone2 = fmt.Sprintf("%s-%s", machineClassNamePool2Zone2, workerPoolHash2)
					)

					addNameAndSecretToMachineClass(machineClassPool1Zone1, machineClassWithHashPool1Zone1, w.Spec.SecretRef)
					addNameAndSecretToMachineClass(machineClassPool1Zone2, machineClassWithHashPool1Zone2, w.Spec.SecretRef)
					addNameAndSecretToMachineClass(machineClassPool2Zone1, machineClassWithHashPool2Zone1, w.Spec.SecretRef)
					addNameAndSecretToMachineClass(machineClassPool2Zone2, machineClassWithHashPool2Zone2, w.Spec.SecretRef)

					machineClasses = map[string]interface{}{"machineClasses": []map[string]interface{}{
						machineClassPool1Zone1,
						machineClassPool1Zone2,
						machineClassPool2Zone1,
						machineClassPool2Zone2,
					}}

					machineDeployments = worker.MachineDeployments{
						{
							Name:                 machineClassNamePool1Zone1,
							ClassName:            machineClassWithHashPool1Zone1,
							SecretName:           machineClassWithHashPool1Zone1,
							Minimum:              worker.DistributeOverZones(0, minPool1, 2),
							Maximum:              worker.DistributeOverZones(0, maxPool1, 2),
							MaxSurge:             worker.DistributePositiveIntOrPercent(0, maxSurgePool1, 2, maxPool1),
							MaxUnavailable:       worker.DistributePositiveIntOrPercent(0, maxUnavailablePool1, 2, minPool1),
							MachineConfiguration: machineConfiguration,
						},
						{
							Name:                 machineClassNamePool1Zone2,
							ClassName:            machineClassWithHashPool1Zone2,
							SecretName:           machineClassWithHashPool1Zone2,
							Minimum:              worker.DistributeOverZones(1, minPool1, 2),
							Maximum:              worker.DistributeOverZones(1, maxPool1, 2),
							MaxSurge:             worker.DistributePositiveIntOrPercent(1, maxSurgePool1, 2, maxPool1),
							MaxUnavailable:       worker.DistributePositiveIntOrPercent(1, maxUnavailablePool1, 2, minPool1),
							MachineConfiguration: machineConfiguration,
						},
						{
							Name:                 machineClassNamePool2Zone1,
							ClassName:            machineClassWithHashPool2Zone1,
							SecretName:           machineClassWithHashPool2Zone1,
							Minimum:              worker.DistributeOverZones(0, minPool2, 2),
							Maximum:              worker.DistributeOverZones(0, maxPool2, 2),
							MaxSurge:             worker.DistributePositiveIntOrPercent(0, maxSurgePool2, 2, maxPool2),
							MaxUnavailable:       worker.DistributePositiveIntOrPercent(0, maxUnavailablePool2, 2, minPool2),
							MachineConfiguration: machineConfiguration,
						},
						{
							Name:                 machineClassNamePool2Zone2,
							ClassName:            machineClassWithHashPool2Zone2,
							SecretName:           machineClassWithHashPool2Zone2,
							Minimum:              worker.DistributeOverZones(1, minPool2, 2),
							Maximum:              worker.DistributeOverZones(1, maxPool2, 2),
							MaxSurge:             worker.DistributePositiveIntOrPercent(1, maxSurgePool2, 2, maxPool2),
							MaxUnavailable:       worker.DistributePositiveIntOrPercent(1, maxUnavailablePool2, 2, minPool2),
							MachineConfiguration: machineConfiguration,
						},
					}

				})

				It("should return the expected machine deployments for profile image types", func() {
					workerDelegate, _ = NewWorkerDelegate(common.NewClientContext(c, scheme, decoder), chartApplier, "", w, cluster)
					gomock.InOrder(
						c.EXPECT().DeleteAllOf(context.TODO(), &machinev1alpha1.AlicloudMachineClass{}, client.InNamespace(namespace)),
						chartApplier.EXPECT().
							Apply(
								context.TODO(),
								filepath.Join(alicloud.InternalChartsPath, "machineclass"),
								namespace,
								"machineclass",
								kubernetes.Values(machineClasses),
							),
					)

					// Test workerDelegate.DeployMachineClasses()
					err := workerDelegate.DeployMachineClasses(context.TODO())
					Expect(err).NotTo(HaveOccurred())

					// Test workerDelegate.UpdateMachineDeployments()
					expectedImages := &apiv1alpha1.WorkerStatus{
						TypeMeta: metav1.TypeMeta{
							APIVersion: apiv1alpha1.SchemeGroupVersion.String(),
							Kind:       "WorkerStatus",
						},
						MachineImages: []apiv1alpha1.MachineImage{
							{
								Name:      machineImageName,
								Version:   machineImageVersion,
								ID:        machineImageID,
								Encrypted: pointer.BoolPtr(false),
							},
							{
								Name:      machineImageName,
								Version:   machineImageVersion,
								ID:        encryptedImageID,
								Encrypted: pointer.BoolPtr(true),
							},
						},
					}

					workerWithExpectedImages := w.DeepCopy()
					workerWithExpectedImages.Status.ProviderStatus = &runtime.RawExtension{
						Object: expectedImages,
					}

					ctx := context.TODO()
					c.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&extensionsv1alpha1.Worker{})).Return(nil)
					c.EXPECT().Status().Return(statusWriter)
					statusWriter.EXPECT().Update(ctx, workerWithExpectedImages).Return(nil)

					err = workerDelegate.UpdateMachineImagesStatus(ctx)
					Expect(err).NotTo(HaveOccurred())

					// Test workerDelegate.GenerateMachineDeployments()
					result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(Equal(machineDeployments))
				})
			})

			It("should return err when the infrastructure provider status cannot be decoded", func() {
				workerDelegate, _ = NewWorkerDelegate(common.NewClientContext(c, scheme, decoder), chartApplier, "", w, cluster)

				// Deliberately setting InfrastructureProviderStatus to empty
				w.Spec.InfrastructureProviderStatus = &runtime.RawExtension{}
				err := workerDelegate.DeployMachineClasses(context.TODO())
				Expect(err).To(HaveOccurred())
			})

			It("should return error when failing to delete AlicloudMachineClass", func() {
				workerDelegate, _ = NewWorkerDelegate(common.NewClientContext(c, scheme, decoder), chartApplier, "", w, cluster)
				c.EXPECT().DeleteAllOf(context.TODO(), &machinev1alpha1.AlicloudMachineClass{}, client.InNamespace(namespace)).Return(fmt.Errorf("fake error"))

				// Test workerDelegate.DeployMachineClasses()
				err := workerDelegate.DeployMachineClasses(context.TODO())
				Expect(err).To(HaveOccurred())
			})

			It("should fail because the version is invalid", func() {
				clusterWithoutImages.Shoot.Spec.Kubernetes.Version = "invalid"
				workerDelegate, _ = NewWorkerDelegate(common.NewClientContext(c, scheme, decoder), chartApplier, "", w, cluster)

				result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should fail because the infrastructure status cannot be decoded", func() {
				w.Spec.InfrastructureProviderStatus = &runtime.RawExtension{}

				workerDelegate, _ = NewWorkerDelegate(common.NewClientContext(c, scheme, decoder), chartApplier, "", w, cluster)

				result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should fail because the security group cannot be found", func() {
				w.Spec.InfrastructureProviderStatus = &runtime.RawExtension{
					Raw: encode(&api.InfrastructureStatus{
						VPC: api.VPCStatus{},
					}),
				}

				workerDelegate, _ = NewWorkerDelegate(common.NewClientContext(c, scheme, decoder), chartApplier, "", w, cluster)

				result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should fail because the machine image cannot be found", func() {
				workerDelegate, _ = NewWorkerDelegate(common.NewClientContext(c, scheme, decoder), chartApplier, "", w, clusterWithoutImages)

				result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should fail because the vswitch id cannot be found", func() {
				w.Spec.InfrastructureProviderStatus = &runtime.RawExtension{
					Raw: encode(&api.InfrastructureStatus{
						VPC: api.VPCStatus{
							VSwitches: []api.VSwitch{},
							SecurityGroups: []api.SecurityGroup{
								{
									ID:      securityGroupID,
									Purpose: "nodes",
								},
							},
						},
					}),
				}

				workerDelegate, _ = NewWorkerDelegate(common.NewClientContext(c, scheme, decoder), chartApplier, "", w, cluster)

				result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should fail because the volume size cannot be decoded", func() {
				w.Spec.Pools[0].Volume.Size = "not-decodeable"

				workerDelegate, _ = NewWorkerDelegate(common.NewClientContext(c, scheme, decoder), chartApplier, "", w, cluster)

				result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("should set expected machineControllerManager settings on machine deployment", func() {
				testDrainTimeout := metav1.Duration{Duration: 10 * time.Minute}
				testHealthTimeout := metav1.Duration{Duration: 20 * time.Minute}
				testCreationTimeout := metav1.Duration{Duration: 30 * time.Minute}
				testMaxEvictRetries := int32(30)
				testNodeConditions := []string{"ReadonlyFilesystem", "KernelDeadlock", "DiskPressure"}
				w.Spec.Pools[0].MachineControllerManagerSettings = &gardencorev1beta1.MachineControllerManagerSettings{
					MachineDrainTimeout:    &testDrainTimeout,
					MachineCreationTimeout: &testCreationTimeout,
					MachineHealthTimeout:   &testHealthTimeout,
					MaxEvictRetries:        &testMaxEvictRetries,
					NodeConditions:         testNodeConditions,
				}

				workerDelegate, _ = NewWorkerDelegate(common.NewClientContext(c, scheme, decoder), chartApplier, "", w, cluster)

				result, err := workerDelegate.GenerateMachineDeployments(context.TODO())
				resultSettings := result[0].MachineConfiguration
				resultNodeConditions := strings.Join(testNodeConditions, ",")

				Expect(err).NotTo(HaveOccurred())
				Expect(resultSettings.MachineDrainTimeout).To(Equal(&testDrainTimeout))
				Expect(resultSettings.MachineCreationTimeout).To(Equal(&testCreationTimeout))
				Expect(resultSettings.MachineHealthTimeout).To(Equal(&testHealthTimeout))
				Expect(resultSettings.MaxEvictRetries).To(Equal(&testMaxEvictRetries))
				Expect(resultSettings.NodeConditions).To(Equal(&resultNodeConditions))
			})
		})
	})
})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}

func useDefaultMachineClass(def map[string]interface{}, keyValues ...interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(def)+1)

	for k, v := range def {
		out[k] = v
	}

	for i := 0; i < len(keyValues); i += 2 {
		out[keyValues[i].(string)] = keyValues[i+1]
	}

	return out
}

func addNameAndSecretToMachineClass(class map[string]interface{}, name string, credentialsSecretRef corev1.SecretReference) {
	class["name"] = name
	class["credentialsSecretRef"] = map[string]interface{}{
		"name":      credentialsSecretRef.Name,
		"namespace": credentialsSecretRef.Namespace,
	}
	class["labels"] = map[string]string{
		v1beta1constants.GardenerPurpose: genericworkeractuator.GardenPurposeMachineClass,
	}
}
