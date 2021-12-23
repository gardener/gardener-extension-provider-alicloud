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

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/install"
	mockalicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/mock/provider-alicloud/alicloud/client"
	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	corev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"

	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"

	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
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

func expectInject(ok bool, err error) {
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(BeTrue(), "no injection happened")
}

func expectEncode(data []byte, err error) []byte {
	Expect(err).NotTo(HaveOccurred())
	Expect(data).NotTo(BeNil())
	return data
}

var _ = Describe("Mutating Shoot", func() {
	var (
		oldShoot              *corev1beta1.Shoot
		newShoot              *corev1beta1.Shoot
		shoot                 extensionswebhook.Mutator
		ctx                   context.Context
		ctrl                  *gomock.Controller
		c                     *mockclient.MockClient
		scheme                *runtime.Scheme
		apiReader             *mockclient.MockReader
		serializer            runtime.Serializer
		alicloudClientFactory *mockalicloudclient.MockClientFactory
		ecsClient             *mockalicloudclient.MockECS
		secretBinding         *corev1beta1.SecretBinding
		secret                *corev1.Secret

		config       *api.CloudProfileConfig
		configYAML   []byte
		cloudProfile *corev1beta1.CloudProfile
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		c = mockclient.NewMockClient(ctrl)
		apiReader = mockclient.NewMockReader(ctrl)

		scheme = runtime.NewScheme()
		install.Install(scheme)
		Expect(controller.AddToScheme(scheme)).To(Succeed())
		serializer = json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{Yaml: true})
		alicloudClientFactory = mockalicloudclient.NewMockClientFactory(ctrl)
		ecsClient = mockalicloudclient.NewMockECS(ctrl)
		ctx = context.TODO()

		shoot = NewShootMutatorWithDeps(alicloudClientFactory)
		expectInject(inject.ClientInto(c, shoot))
		expectInject(inject.SchemeInto(scheme, shoot))
		expectInject(inject.APIReaderInto(apiReader, shoot))

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

		configYAML = expectEncode(runtime.Encode(serializer, config))
		cloudProfile = &corev1beta1.CloudProfile{
			Spec: corev1beta1.CloudProfileSpec{
				ProviderConfig: &runtime.RawExtension{
					Raw: configYAML,
				},
			},
		}
		oldShoot = &corev1beta1.Shoot{
			Spec: corev1beta1.ShootSpec{
				Provider: corev1beta1.Provider{
					Workers: []corev1beta1.Worker{
						{
							Machine: corev1beta1.Machine{
								Image: &corev1beta1.ShootMachineImage{
									Name:    imageName,
									Version: pointer.StringPtr(imageVersionStr),
								},
							},
							Volume: &corev1beta1.Volume{
								Encrypted: pointer.BoolPtr(true),
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
				SecretBindingName: name,
				Provider: corev1beta1.Provider{
					Workers: []corev1beta1.Worker{
						{
							Machine: corev1beta1.Machine{
								Image: &corev1beta1.ShootMachineImage{
									Name:    imageName,
									Version: pointer.StringPtr(imageVersionStr),
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
									Version: pointer.StringPtr(imageVersionStr),
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
	Context("#Encrypted System Disk", func() {
		It("should set encrypted flag as true for new shoot ", func() {
			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key("alicloud"), gomock.AssignableToTypeOf(&corev1beta1.CloudProfile{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.CloudProfile) error {
						*obj = *cloudProfile
						return nil
					},
				),
				c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1beta1.SecretBinding{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.SecretBinding) error {
						*obj = *secretBinding
						return nil
					},
				),
				apiReader.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret) error {
						*obj = *secret
						return nil
					},
				),

				alicloudClientFactory.EXPECT().NewECSClient(regionId, accessKeyID, accessKeySecret).Return(ecsClient, nil),
				ecsClient.EXPECT().CheckIfImageExists(ctx, imageId).Return(false, nil),
				//ecsClient.EXPECT().CheckIfImageOwnedByAliCloud(imageId).Return(false, nil)
			)
			err := shoot.Mutate(ctx, newShoot, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(*newShoot.Spec.Provider.Workers[0].Volume.Encrypted).To(BeTrue())
			Expect(*newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted).To(BeTrue())
			Expect(controllerutils.HasTask(newShoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)).To(BeFalse())
		})
		It("should set encrypted flag as false for system disk if image is owned by alicloud", func() {
			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key("alicloud"), gomock.AssignableToTypeOf(&corev1beta1.CloudProfile{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.CloudProfile) error {
						*obj = *cloudProfile
						return nil
					},
				),
				c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1beta1.SecretBinding{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.SecretBinding) error {
						*obj = *secretBinding
						return nil
					},
				),
				apiReader.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret) error {
						*obj = *secret
						return nil
					},
				),

				alicloudClientFactory.EXPECT().NewECSClient(regionId, accessKeyID, accessKeySecret).Return(ecsClient, nil),
				ecsClient.EXPECT().CheckIfImageExists(ctx, imageId).Return(true, nil),
				ecsClient.EXPECT().CheckIfImageOwnedByAliCloud(imageId).Return(true, nil),
			)
			err := shoot.Mutate(ctx, newShoot, nil)
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
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.CloudProfile) error {
						*obj = *cloudProfile
						return nil
					},
				),
				c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1beta1.SecretBinding{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1beta1.SecretBinding) error {
						*obj = *secretBinding
						return nil
					},
				),
				apiReader.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret) error {
						*obj = *secret
						return nil
					},
				),

				alicloudClientFactory.EXPECT().NewECSClient(regionId, accessKeyID, accessKeySecret).Return(ecsClient, nil),
				ecsClient.EXPECT().CheckIfImageExists(ctx, imageId).Return(false, nil),
			)
			err := shoot.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(*newShoot.Spec.Provider.Workers[1].Volume.Encrypted).To(BeTrue())
			Expect(*newShoot.Spec.Provider.Workers[1].DataVolumes[0].Encrypted).To(BeTrue())
			Expect(*newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted).To(BeTrue())
		})
		It("Should Keep encrypted flag unchanged if shoot was created in old version and this flag is not set explictly", func() {
			oldShoot.Spec.Provider.Workers[0].Volume.Encrypted = nil
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = nil
			sameName := "worker1"
			oldShoot.Spec.Provider.Workers[0].Name = sameName
			newShoot.Spec.Provider.Workers[0].Name = sameName

			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = sameName
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = sameName
			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = nil
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = nil

			err := shoot.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(newShoot.Spec.Provider.Workers[0].Volume.Encrypted == nil).To(BeTrue())
			Expect(newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted == nil).To(BeTrue())

		})
		It("Should keep default encrypted flag unchanged if shoot is created in new version and this flag is not set explictly", func() {
			oldShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.BoolPtr(true)
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = nil
			sameName := "worker1"
			oldShoot.Spec.Provider.Workers[0].Name = sameName
			newShoot.Spec.Provider.Workers[0].Name = sameName

			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = pointer.BoolPtr(true)
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = nil
			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = sameName
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = sameName

			err := shoot.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(*newShoot.Spec.Provider.Workers[0].Volume.Encrypted).To(BeTrue())
			Expect(*newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted).To(BeTrue())

		})
		It("Should use set encrypted flag if it's specified in new shoot", func() {
			oldShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.BoolPtr(true)
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.BoolPtr(false)
			sameName := "worker1"
			oldShoot.Spec.Provider.Workers[0].Name = sameName
			newShoot.Spec.Provider.Workers[0].Name = sameName

			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = pointer.BoolPtr(false)
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted = pointer.BoolPtr(true)
			oldShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = sameName
			newShoot.Spec.Provider.Workers[0].DataVolumes[0].Name = sameName

			err := shoot.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(*newShoot.Spec.Provider.Workers[0].Volume.Encrypted).To(BeFalse())
			Expect(*newShoot.Spec.Provider.Workers[0].DataVolumes[0].Encrypted).To(BeTrue())

		})
		It("should not reconcile infra if no system disk is encrypted", func() {
			err := shoot.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(controllerutils.HasTask(newShoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)).To(BeFalse())
		})

		It("should not reconcile infra if system disk is already encrypted", func() {
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.BoolPtr(true)
			err := shoot.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(controllerutils.HasTask(newShoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)).To(BeFalse())
		})

		It("should not reconcile infra if new version of machine is not encrypted", func() {
			newShoot.Spec.Provider.Workers[1].Machine.Image.Version = pointer.StringPtr("2.0")
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.BoolPtr(true)
			err := shoot.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(controllerutils.HasTask(newShoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)).To(BeFalse())
		})

		It("should reconcile infra if new version of machine is added and it is encrypted", func() {
			newShoot.Spec.Provider.Workers[0].Machine.Image.Version = pointer.StringPtr("2.0")
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.BoolPtr(true)
			err := shoot.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(controllerutils.HasTask(newShoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)).To(BeTrue())
		})

		It("should reconcile infra if machine is changed to be encrypted", func() {
			oldShoot.Spec.Provider.Workers[0].Volume.Encrypted = nil
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.BoolPtr(true)
			err := shoot.Mutate(ctx, newShoot, oldShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(controllerutils.HasTask(newShoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)).To(BeTrue())
		})
	})
})
