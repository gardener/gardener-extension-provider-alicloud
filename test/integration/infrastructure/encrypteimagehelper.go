// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/retry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client/ros"
	alicloudapi "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	alicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/common"
)

var imageName = "ubuntu"
var imageVersion = "20.04"

func newCluster(namespace string) (*extensionsv1alpha1.Cluster, error) {
	providerConfig := &alicloudv1alpha1.CloudProfileConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CloudProfileConfig",
			APIVersion: alicloudv1alpha1.SchemeGroupVersion.String(),
		},
		MachineImages: []alicloudv1alpha1.MachineImages{
			{
				Name: imageName,
				Versions: []alicloudv1alpha1.MachineImageVersion{
					{
						Version: imageVersion,
						Regions: []alicloudv1alpha1.RegionIDMapping{
							{
								Name: *region,
								ID:   getImageId(*region),
							},
						},
					},
				},
			},
		},
	}
	providerConfigJSON, err := json.Marshal(providerConfig)
	if err != nil {
		return nil, err
	}

	cloudProfile := &gardencorev1beta1.CloudProfile{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CloudProfile",
			APIVersion: gardencorev1beta1.SchemeGroupVersion.String(),
		},
		Spec: gardencorev1beta1.CloudProfileSpec{
			ProviderConfig: &runtime.RawExtension{
				Raw: providerConfigJSON,
			},
		},
	}
	cloudProfileJSON, err := json.Marshal(cloudProfile)
	if err != nil {
		return nil, err
	}

	shoot := &gardencorev1beta1.Shoot{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Shoot",
			APIVersion: gardencorev1beta1.SchemeGroupVersion.String(),
		},
		Spec: gardencorev1beta1.ShootSpec{
			Provider: gardencorev1beta1.Provider{
				Type: "alicloud",
				Workers: []gardencorev1beta1.Worker{
					{
						Machine: gardencorev1beta1.Machine{
							Type: "ecs.g6.2xlarge",
							Image: &gardencorev1beta1.ShootMachineImage{
								Name:    imageName,
								Version: ptr.To(imageVersion),
							},
						},
						Volume: &gardencorev1beta1.Volume{
							Name:       ptr.To("workgroup"),
							Type:       ptr.To("cloud_efficiency"),
							VolumeSize: "200Gi",
							Encrypted:  enableEncryptedImage,
						},
					},
				},
			},
			Networking: &gardencorev1beta1.Networking{
				Pods: ptr.To(podCIDR),
			},
		},
	}
	shootJSON, err := json.Marshal(shoot)
	if err != nil {
		return nil, err
	}

	cluster := &extensionsv1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: extensionsv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
		Spec: extensionsv1alpha1.ClusterSpec{
			CloudProfile: runtime.RawExtension{
				Raw: cloudProfileJSON,
			},
			Seed: runtime.RawExtension{
				Raw: []byte("{}"),
			},
			Shoot: runtime.RawExtension{
				Raw: shootJSON,
			},
		},
	}

	return cluster, nil
}

func deleteEncryptedImageStackIfExists(ctx context.Context, clientFactory alicloudclient.ClientFactory) error {
	stackName := common.GetEncryptImageStackName(imageName, imageVersion, *region)
	listRequest := ros.CreateListStacksRequest()
	listRequest.StackName = &[]string{stackName}
	listRequest.RegionId = *region
	listRequest.SetScheme("HTTPS")

	rosClient, err := clientFactory.NewROSClient(*region, *accessKeyID, *accessKeySecret)
	if err != nil {
		return err
	}

	response, err := rosClient.ListStacks(listRequest)
	if err != nil {
		return err
	}

	if len(response.Stacks) == 0 {
		return nil
	}

	deleteRequest := ros.CreateDeleteStackRequest()
	deleteRequest.StackId = response.Stacks[0].StackId
	deleteRequest.RegionId = *region

	if _, err := rosClient.DeleteStack(deleteRequest); err != nil {
		return err
	}

	return retry.UntilTimeout(ctx, 5*time.Second, 5*time.Minute, func(_ context.Context) (done bool, err error) {
		response, err := rosClient.ListStacks(listRequest)
		if err != nil {
			return retry.MinorError(err)
		}
		if len(response.Stacks) == 0 {
			return retry.Ok()
		}
		return retry.MinorError(nil)
	})
}

func verifyStackExists(ctx context.Context, clientFactory alicloudclient.ClientFactory) error {
	stackName := common.GetEncryptImageStackName(imageName, imageVersion, *region)
	listRequest := ros.CreateListStacksRequest()
	listRequest.StackName = &[]string{stackName}
	listRequest.RegionId = *region
	listRequest.SetScheme("HTTPS")

	rosClient, err := clientFactory.NewROSClient(*region, *accessKeyID, *accessKeySecret)
	if err != nil {
		return err
	}

	// It will wait until both infra is provisioned and image is prepared
	return retry.UntilTimeout(ctx, 10*time.Second, 50*time.Minute, func(_ context.Context) (done bool, err error) {
		listStacksResponse, err := rosClient.ListStacks(listRequest)
		if err != nil {
			return retry.MinorError(err)
		}
		if len(listStacksResponse.Stacks) == 0 {
			return retry.MinorError(fmt.Errorf("stack %s doesn't exist", stackName))
		}

		getStackRequest := ros.CreateGetStackRequest()
		getStackRequest.StackId = listStacksResponse.Stacks[0].StackId
		getStackRequest.SetScheme("HTTPS")

		getStackResponse, err := rosClient.GetStack(getStackRequest)
		if err != nil {
			if serverErr, ok := err.(*errors.ServerError); ok {
				// Always, if it is not server error, we should not retry
				if serverErr.HttpStatus() < 500 {
					return retry.SevereError(err)
				}
			}
			return retry.MinorError(err)
		}

		if getStackResponse.Status == "CREATE_COMPLETE" {
			return retry.Ok()
		}

		return retry.MinorError(nil)
	})
}

func verifyImageInfraStatus(status *alicloudv1alpha1.InfrastructureStatus) error {
	var machineImages []alicloudapi.MachineImage

	scheme := runtime.NewScheme()
	if err := alicloudapi.AddToScheme(scheme); err != nil {
		return err
	}

	if err := alicloudv1alpha1.AddToScheme(scheme); err != nil {
		return err
	}

	for _, img := range status.MachineImages {
		converted := &alicloudapi.MachineImage{}
		if err := scheme.Convert(&img, converted, nil); err != nil {
			return err
		}
		machineImages = append(machineImages, *converted)
	}

	_, err := helper.FindMachineImage(machineImages, imageName, imageVersion, *enableEncryptedImage)
	return err
}
