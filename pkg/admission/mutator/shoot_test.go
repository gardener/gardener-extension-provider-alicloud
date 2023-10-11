// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package mutator

import (
	"context"
	"time"

	calicov1alpha1 "github.com/gardener/gardener-extension-networking-calico/pkg/apis/calico/v1alpha1"
	ciliumv1alpha1 "github.com/gardener/gardener-extension-networking-cilium/pkg/apis/cilium/v1alpha1"
	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	corev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/controllerutils"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	mockmanager "github.com/gardener/gardener/pkg/mock/controller-runtime/manager"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/install"
	apisalicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	mockalicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/mock/provider-alicloud/alicloud/client"
)

const (
	name      = "alicloud"
	namespace = "garden"

	regionId        = "cn-shanghai"
	accessKeyID     = "accessKeyID"
	accessKeySecret = "accessKeySecret"

	imageName       = "gardenlinux"
	imageVersionStr = "318.9.0"
	imageId         = "m-uf6htf9lstsi99xr2out"
)

func expectEncode(data []byte, err error) []byte {
	Expect(err).NotTo(HaveOccurred())
	Expect(data).NotTo(BeNil())
	return data
}

var _ = Describe("Mutating Shoot", func() {
	var (
		oldShoot              *corev1beta1.Shoot
		newShoot              *corev1beta1.Shoot
		mutator               extensionswebhook.Mutator
		ctx                   context.Context
		ctrl                  *gomock.Controller
		c                     *mockclient.MockClient
		mgr                   *mockmanager.MockManager
		scheme                *runtime.Scheme
		apiReader             *mockclient.MockReader
		serializer            runtime.Serializer
		alicloudClientFactory *mockalicloudclient.MockClientFactory
		ecsClient             *mockalicloudclient.MockECS
		secretBinding         *corev1beta1.SecretBinding
		secret                *corev1.Secret
		now                   = metav1.Now()

		config       *api.CloudProfileConfig
		configJson   []byte
		cloudProfile *corev1beta1.CloudProfile
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		c = mockclient.NewMockClient(ctrl)
		apiReader = mockclient.NewMockReader(ctrl)

		scheme = runtime.NewScheme()
		install.Install(scheme)
		Expect(controller.AddToScheme(scheme)).To(Succeed())

		mgr = mockmanager.NewMockManager(ctrl)
		mgr.EXPECT().GetClient().Return(c)
		mgr.EXPECT().GetScheme().Return(scheme).Times(3)
		mgr.EXPECT().GetAPIReader().Return(apiReader)

		serializer = json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{})
		alicloudClientFactory = mockalicloudclient.NewMockClientFactory(ctrl)
		ecsClient = mockalicloudclient.NewMockECS(ctrl)
		ctx = context.TODO()

		mutator = NewShootMutatorWithDeps(mgr, alicloudClientFactory)

		secretBinding = &corev1beta1.SecretBinding{
			SecretRef: corev1.SecretReference{
				Name:      name,
				Namespace: namespace,
			},
		}

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				alicloud.AccessKeyID:     []byte(accessKeyID),
				alicloud.AccessKeySecret: []byte(accessKeySecret),
			},
		}

		config = &api.CloudProfileConfig{
			MachineImages: []api.MachineImages{
				{
					Name: imageName,
					Versions: []api.MachineImageVersion{
						{
							Version: imageVersionStr,
							Regions: []api.RegionIDMapping{
								{
									Name: regionId,
									ID:   imageId,
								},
							},
						},
					},
				},
			},
		}

		configJson = expectEncode(runtime.Encode(serializer, config))
		cloudProfile = &corev1beta1.CloudProfile{
			Spec: corev1beta1.CloudProfileSpec{
				ProviderConfig: &runtime.RawExtension{
					Raw: configJson,
				},
			},
		}
		controlPlaneConfig := &apisalicloudv1alpha1.ControlPlaneConfig{
			CSI: &apisalicloudv1alpha1.CSI{
				EnableADController: pointer.Bool(false),
			}}
		oldShoot = &corev1beta1.Shoot{
			Spec: corev1beta1.ShootSpec{
				SeedName: pointer.String("alicloud"),
				Networking: &corev1beta1.Networking{
					Nodes: pointer.String("10.250.0.0/16"),
					Type:  pointer.String("calico"),
				},
				Provider: corev1beta1.Provider{
					Type: alicloud.Type,
					ControlPlaneConfig: &runtime.RawExtension{
						Raw: expectEncode(
							runtime.Encode(serializer, controlPlaneConfig))},
					Workers: []corev1beta1.Worker{
						{
							Machine: corev1beta1.Machine{
								Image: &corev1beta1.ShootMachineImage{
									Name:    imageName,
									Version: pointer.String(imageVersionStr),
								},
							},
							Volume: &corev1beta1.Volume{
								Encrypted: pointer.Bool(true),
							},
							DataVolumes: []corev1beta1.DataVolume{
								{},
							},
						},
					},
				},
			},
		}

		newShoot = &corev1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: corev1beta1.ShootSpec{
				SeedName: pointer.String("alicloud"),
				Networking: &corev1beta1.Networking{
					Nodes: pointer.String("10.250.0.0/16"),
					Type:  pointer.String("calico"),
				},
				SecretBindingName: pointer.String(name),
				Provider: corev1beta1.Provider{
					Type: alicloud.Type,
					Workers: []corev1beta1.Worker{
						{
							Machine: corev1beta1.Machine{
								Image: &corev1beta1.ShootMachineImage{
									Name:    imageName,
									Version: pointer.String(imageVersionStr),
								},
							},
							Volume: &corev1beta1.Volume{},
							DataVolumes: []corev1beta1.DataVolume{
								{},
							},
						},
						{
							Machine: corev1beta1.Machine{
								Image: &corev1beta1.ShootMachineImage{
									Name:    imageName,
									Version: pointer.String(imageVersionStr),
								},
							},
						},
					},
				},
				CloudProfileName: "alicloud",
				Region:           regionId,
			},
		}
	})
	AfterEach(func() {
		ctrl.Finish()
	})
	Context("#ControlPlaneConfig", func() {
		It("should default EnableADController true if EnableADController is not set when creating a shoot ", func() {
			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key("alicloud"), gomock.AssignableToTypeOf(&corev1beta1.CloudProfile{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.CloudProfile, _ ...client.GetOption) error {
						*obj = *cloudProfile
						return nil
					},
				),
				c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1beta1.SecretBinding{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.SecretBinding, _ ...client.GetOption) error {
						*obj = *secretBinding
						return nil
					},
				),
				apiReader.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
						*obj = *secret
						return nil
					},
				),

				alicloudClientFactory.EXPECT().NewECSClient(regionId, accessKeyID, accessKeySecret).Return(ecsClient, nil),
				ecsClient.EXPECT().CheckIfImageExists(imageId).Return(false, nil),
			)
			err := mutator.Mutate(ctx, newShoot, nil)
			Expect(err).NotTo(HaveOccurred())
			cpConfig := &apisalicloudv1alpha1.ControlPlaneConfig{}
			_, _, err = serializer.Decode(newShoot.Spec.Provider.ControlPlaneConfig.Raw, nil, cpConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(cpConfig.CSI).NotTo(BeNil())
			Expect(cpConfig.CSI.EnableADController).NotTo(BeNil())
			Expect(*cpConfig.CSI.EnableADController).To(BeTrue())
		})
		It("should keep old EnableADController if EnableADController is not set when update a shoot ", func() {
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			cpConfig := &apisalicloudv1alpha1.ControlPlaneConfig{}
			_, _, err = serializer.Decode(newShoot.Spec.Provider.ControlPlaneConfig.Raw, nil, cpConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(cpConfig.CSI).NotTo(BeNil())
			Expect(cpConfig.CSI.EnableADController).NotTo(BeNil())
			Expect(*cpConfig.CSI.EnableADController).To(BeFalse())
		})
	})
	Context("#Encrypted System Disk", func() {
		It("should set encrypted flag as true for new shoot ", func() {
			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key("alicloud"), gomock.AssignableToTypeOf(&corev1beta1.CloudProfile{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.CloudProfile, _ ...client.GetOption) error {
						*obj = *cloudProfile
						return nil
					},
				),
				c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1beta1.SecretBinding{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.SecretBinding, _ ...client.GetOption) error {
						*obj = *secretBinding
						return nil
					},
				),
				apiReader.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
						*obj = *secret
						return nil
					},
				),

				alicloudClientFactory.EXPECT().NewECSClient(regionId, accessKeyID, accessKeySecret).Return(ecsClient, nil),
				ecsClient.EXPECT().CheckIfImageExists(imageId).Return(false, nil),
				//ecsClient.EXPECT().CheckIfImageOwnedByAliCloud(imageId).Return(false, nil)
			)
			err := mutator.Mutate(ctx, newShoot, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(*newShoot.Spec.Provider.Workers[0].Volume.Encrypted).To(BeTrue())
			Expect(*newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted).To(BeTrue())
			Expect(controllerutils.HasTask(newShoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)).To(BeFalse())
		})
		It("should set encrypted flag as false for system disk if image is owned by alicloud", func() {
			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key("alicloud"), gomock.AssignableToTypeOf(&corev1beta1.CloudProfile{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.CloudProfile, _ ...client.GetOption) error {
						*obj = *cloudProfile
						return nil
					},
				),
				c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1beta1.SecretBinding{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.SecretBinding, _ ...client.GetOption) error {
						*obj = *secretBinding
						return nil
					},
				),
				apiReader.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
						*obj = *secret
						return nil
					},
				),

				alicloudClientFactory.EXPECT().NewECSClient(regionId, accessKeyID, accessKeySecret).Return(ecsClient, nil),
				ecsClient.EXPECT().CheckIfImageExists(imageId).Return(true, nil),
				ecsClient.EXPECT().CheckIfImageOwnedByAliCloud(imageId).Return(true, nil),
			)
			err := mutator.Mutate(ctx, newShoot, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot.Spec.Provider.Workers[0].Volume.Encrypted == nil).To(BeTrue())
			Expect(*newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted).To(BeTrue())
		})
		It("should set encrypted flag as true for newly added worker or datavolume", func() {
			sameName := "worker1"
			newName := "newWorker"

			oldShoot.Spec.Provider.Workers[0].Volume.Encrypted = nil
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = nil
			oldShoot.Spec.Provider.Workers[0].Name = sameName
			newShoot.Spec.Provider.Workers[0].Name = sameName

			newShoot.Spec.Provider.Workers[1].Name = newName
			newShoot.Spec.Provider.Workers[1].Volume = &corev1beta1.Volume{}
			newShoot.Spec.Provider.Workers[1].DataVolumes = []corev1beta1.DataVolume{{}}
			//simulate to add datavolume disk
			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = "old"
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = newName
			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = nil
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = nil

			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key("alicloud"), gomock.AssignableToTypeOf(&corev1beta1.CloudProfile{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.CloudProfile, _ ...client.GetOption) error {
						*obj = *cloudProfile
						return nil
					},
				),
				c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1beta1.SecretBinding{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.SecretBinding, _ ...client.GetOption) error {
						*obj = *secretBinding
						return nil
					},
				),
				apiReader.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
						*obj = *secret
						return nil
					},
				),

				alicloudClientFactory.EXPECT().NewECSClient(regionId, accessKeyID, accessKeySecret).Return(ecsClient, nil),
				ecsClient.EXPECT().CheckIfImageExists(imageId).Return(false, nil),
			)
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(*newShoot.Spec.Provider.Workers[1].Volume.Encrypted).To(BeTrue())
			Expect(*newShoot.Spec.Provider.Workers[1].DataVolumes[0].Encrypted).To(BeTrue())
			Expect(*newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted).To(BeTrue())
		})
		It("Should Keep encrypted flag unchanged if shoot was created in old version and this flag is not set explicitly", func() {
			oldShoot.Spec.Provider.Workers[0].Volume.Encrypted = nil
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = nil
			sameName := "worker1"
			oldShoot.Spec.Provider.Workers[0].Name = sameName
			newShoot.Spec.Provider.Workers[0].Name = sameName

			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = sameName
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = sameName
			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = nil
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = nil

			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot.Spec.Provider.Workers[0].Volume.Encrypted == nil).To(BeTrue())
			Expect(newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted == nil).To(BeTrue())

		})
		It("Should keep default encrypted flag unchanged if shoot is created in new version and this flag is not set explicitly", func() {
			oldShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.Bool(true)
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = nil
			sameName := "worker1"
			oldShoot.Spec.Provider.Workers[0].Name = sameName
			newShoot.Spec.Provider.Workers[0].Name = sameName

			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = pointer.Bool(true)
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = nil
			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = sameName
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = sameName

			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(*newShoot.Spec.Provider.Workers[0].Volume.Encrypted).To(BeTrue())
			Expect(*newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted).To(BeTrue())

		})
		It("Should use set encrypted flag if it's specified in new shoot", func() {
			oldShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.Bool(true)
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.Bool(false)
			sameName := "worker1"
			oldShoot.Spec.Provider.Workers[0].Name = sameName
			newShoot.Spec.Provider.Workers[0].Name = sameName

			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = pointer.Bool(false)
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = pointer.Bool(true)
			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = sameName
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = sameName

			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(*newShoot.Spec.Provider.Workers[0].Volume.Encrypted).To(BeFalse())
			Expect(*newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted).To(BeTrue())

		})
		It("should not reconcile infra if no system disk is encrypted", func() {
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(controllerutils.HasTask(newShoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)).To(BeFalse())
		})

		It("should not reconcile infra if system disk is already encrypted", func() {
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.Bool(true)
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(controllerutils.HasTask(newShoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)).To(BeFalse())
		})

		It("should not reconcile infra if new version of machine is not encrypted", func() {
			newShoot.Spec.Provider.Workers[1].Machine.Image.Version = pointer.String("2.0")
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.Bool(true)
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(controllerutils.HasTask(newShoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)).To(BeFalse())
		})

		It("should reconcile infra if new version of machine is added and it is encrypted", func() {
			newShoot.Spec.Provider.Workers[0].Machine.Image.Version = pointer.String("2.0")
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.Bool(true)
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(controllerutils.HasTask(newShoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)).To(BeTrue())
		})

		It("should reconcile infra if machine is changed to be encrypted", func() {
			oldShoot.Spec.Provider.Workers[0].Volume.Encrypted = nil
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.Bool(true)
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(controllerutils.HasTask(newShoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)).To(BeTrue())
		})

	})

	Context("Workerless Shoot", func() {
		BeforeEach(func() {
			newShoot.Spec.Provider.Workers = nil
		})

		It("should return without mutation when shoot is in scheduled to new seed phase", func() {
			shootExpected := newShoot.DeepCopy()
			err := mutator.Mutate(ctx, newShoot, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot).To(DeepEqual(shootExpected))
		})
	})

	Context("Mutate shoot networking providerconfig for type calico", func() {

		It("should return without mutation when shoot is in scheduled to new seed phase", func() {
			newShoot.Status.LastOperation = &corev1beta1.LastOperation{
				Description:    "test",
				LastUpdateTime: metav1.Time{Time: metav1.Now().Add(time.Second * -1000)},
				Progress:       0,
				Type:           corev1beta1.LastOperationTypeReconcile,
				State:          corev1beta1.LastOperationStateProcessing,
			}
			newShoot.Status.SeedName = pointer.String("aws")
			expectedShootNetworkingProviderConfig := newShoot.Spec.Networking.ProviderConfig.DeepCopy()
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot.Spec.Networking.ProviderConfig).To(DeepEqual(expectedShootNetworkingProviderConfig))
		})

		It("should return without mutation when shoot is in migration or restore phase", func() {
			newShoot.Status.LastOperation = &corev1beta1.LastOperation{
				Description:    "test",
				LastUpdateTime: metav1.Time{Time: metav1.Now().Add(time.Second * -1000)},
				Progress:       0,
				Type:           corev1beta1.LastOperationTypeMigrate,
				State:          corev1beta1.LastOperationStateProcessing,
			}
			newShoot.Status.SeedName = pointer.String("alicloud")
			expectedShootNetworkingProviderConfig := newShoot.Spec.Networking.ProviderConfig.DeepCopy()
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot.Spec.Networking.ProviderConfig).To(DeepEqual(expectedShootNetworkingProviderConfig))
		})

		It("should return without mutation when shoot is in deletion phase", func() {
			newShoot.DeletionTimestamp = &now
			expectedShootNetworkingProviderConfig := newShoot.Spec.Networking.ProviderConfig.DeepCopy()
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot.Spec.Networking.ProviderConfig).To(DeepEqual(expectedShootNetworkingProviderConfig))
		})

		It("should return without mutation when shoot specs have not changed", func() {
			shootWithAnnotations := newShoot.DeepCopy()
			shootWithAnnotations.Annotations = map[string]string{"foo": "bar"}
			shootExpected := shootWithAnnotations.DeepCopy()

			err := mutator.Mutate(ctx, shootWithAnnotations, newShoot)
			Expect(err).To(BeNil())
			Expect(shootWithAnnotations).To(DeepEqual(shootExpected))
		})

		It("should disable overlay for a new shoot", func() {
			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key("alicloud"), gomock.AssignableToTypeOf(&corev1beta1.CloudProfile{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.CloudProfile, _ ...client.GetOption) error {
						*obj = *cloudProfile
						return nil
					},
				),
				c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1beta1.SecretBinding{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.SecretBinding, _ ...client.GetOption) error {
						*obj = *secretBinding
						return nil
					},
				),
				apiReader.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
						*obj = *secret
						return nil
					},
				),

				alicloudClientFactory.EXPECT().NewECSClient(regionId, accessKeyID, accessKeySecret).Return(ecsClient, nil),
				ecsClient.EXPECT().CheckIfImageExists(imageId).Return(true, nil),
				ecsClient.EXPECT().CheckIfImageOwnedByAliCloud(imageId).Return(true, nil),
			)
			err := mutator.Mutate(ctx, newShoot, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot.Spec.Networking.ProviderConfig).To(Equal(&runtime.RawExtension{
				Object: &calicov1alpha1.NetworkConfig{
					Overlay: &calicov1alpha1.Overlay{
						Enabled: false,
					},
					SnatToUpstreamDNS: &calicov1alpha1.SnatToUpstreamDNS{Enabled: false},
				},
			}))
		})

		It("should take overlay field value from old shoot when unspecified in new shoot", func() {
			oldShoot.Spec.Networking.ProviderConfig = &runtime.RawExtension{
				Raw: []byte(`{"overlay":{"enabled":true}}`),
				Object: &calicov1alpha1.NetworkConfig{
					Overlay: &calicov1alpha1.Overlay{
						Enabled: true,
					},
				},
			}
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot.Spec.Networking.ProviderConfig).To(Equal(&runtime.RawExtension{
				Object: &calicov1alpha1.NetworkConfig{
					Overlay: &calicov1alpha1.Overlay{
						Enabled: true,
					},
				},
			}))
		})
	})

	Context("Mutate shoot networking providerconfig for type cilium", func() {

		BeforeEach(func() {
			newShoot.Spec.Networking.Type = pointer.String("cilium")
			oldShoot.Spec.Networking.Type = pointer.String("cilium")
		})

		It("should return without mutation when shoot is in scheduled to new seed phase", func() {
			newShoot.Status.LastOperation = &corev1beta1.LastOperation{
				Description:    "test",
				LastUpdateTime: metav1.Time{Time: metav1.Now().Add(time.Second * -1000)},
				Progress:       0,
				Type:           corev1beta1.LastOperationTypeReconcile,
				State:          corev1beta1.LastOperationStateProcessing,
			}
			newShoot.Status.SeedName = pointer.String("aws")
			expectedShootNetworkingProviderConfig := newShoot.Spec.Networking.ProviderConfig.DeepCopy()
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot.Spec.Networking.ProviderConfig).To(DeepEqual(expectedShootNetworkingProviderConfig))
		})

		It("should return without mutation when shoot is in migration or restore phase", func() {
			newShoot.Status.LastOperation = &corev1beta1.LastOperation{
				Description:    "test",
				LastUpdateTime: metav1.Time{Time: metav1.Now().Add(time.Second * -1000)},
				Progress:       0,
				Type:           corev1beta1.LastOperationTypeMigrate,
				State:          corev1beta1.LastOperationStateProcessing,
			}
			newShoot.Status.SeedName = pointer.String("alicloud")
			expectedShootNetworkingProviderConfig := newShoot.Spec.Networking.ProviderConfig.DeepCopy()
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot.Spec.Networking.ProviderConfig).To(DeepEqual(expectedShootNetworkingProviderConfig))
		})

		It("should return without mutation when shoot is in deletion phase", func() {
			newShoot.DeletionTimestamp = &now
			expectedShootNetworkingProviderConfig := newShoot.Spec.Networking.ProviderConfig.DeepCopy()
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot.Spec.Networking.ProviderConfig).To(DeepEqual(expectedShootNetworkingProviderConfig))
		})

		It("should return without mutation when shoot specs have not changed", func() {
			shootWithAnnotations := newShoot.DeepCopy()
			shootWithAnnotations.Annotations = map[string]string{"foo": "bar"}
			shootExpected := shootWithAnnotations.DeepCopy()

			err := mutator.Mutate(ctx, shootWithAnnotations, newShoot)
			Expect(err).To(BeNil())
			Expect(shootWithAnnotations).To(DeepEqual(shootExpected))
		})

		It("should disable overlay for a new shoot", func() {
			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key("alicloud"), gomock.AssignableToTypeOf(&corev1beta1.CloudProfile{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.CloudProfile, _ ...client.GetOption) error {
						*obj = *cloudProfile
						return nil
					},
				),
				c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1beta1.SecretBinding{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.SecretBinding, _ ...client.GetOption) error {
						*obj = *secretBinding
						return nil
					},
				),
				apiReader.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
						*obj = *secret
						return nil
					},
				),

				alicloudClientFactory.EXPECT().NewECSClient(regionId, accessKeyID, accessKeySecret).Return(ecsClient, nil),
				ecsClient.EXPECT().CheckIfImageExists(imageId).Return(true, nil),
				ecsClient.EXPECT().CheckIfImageOwnedByAliCloud(imageId).Return(true, nil),
			)
			err := mutator.Mutate(ctx, newShoot, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot.Spec.Networking.ProviderConfig).To(Equal(&runtime.RawExtension{
				Object: &ciliumv1alpha1.NetworkConfig{
					Overlay: &ciliumv1alpha1.Overlay{
						Enabled: false,
					},
				},
			}))
		})

		It("should take overlay field value from old shoot when unspecified in new shoot", func() {
			oldShoot.Spec.Networking.ProviderConfig = &runtime.RawExtension{
				Raw: []byte(`{"overlay":{"enabled":true}}`),
				Object: &ciliumv1alpha1.NetworkConfig{
					Overlay: &ciliumv1alpha1.Overlay{
						Enabled: true,
					},
				},
			}
			err := mutator.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot.Spec.Networking.ProviderConfig).To(Equal(&runtime.RawExtension{
				Object: &ciliumv1alpha1.NetworkConfig{
					Overlay: &ciliumv1alpha1.Overlay{
						Enabled: true,
					},
				},
			}))
		})
	})
})
