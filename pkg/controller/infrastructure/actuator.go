// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"
	"strings"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	"github.com/gardener/gardener/extensions/pkg/terraformer"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	alicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/common"
)

// StatusTypeMeta is the TypeMeta of InfrastructureStatus.
var StatusTypeMeta = func() metav1.TypeMeta {
	apiVersion, kind := alicloudv1alpha1.SchemeGroupVersion.WithKind(extensioncontroller.UnsafeGuessKind(&alicloudv1alpha1.InfrastructureStatus{})).ToAPIVersionAndKind()
	return metav1.TypeMeta{
		APIVersion: apiVersion,
		Kind:       kind,
	}
}()

// NewActuator instantiates an actuator with the default dependencies.
func NewActuator(mgr manager.Manager, machineImageOwnerSecretRef *corev1.SecretReference, toBeSharedImageIDs []string, disableProjectedTokenMount bool) (infrastructure.Actuator, error) {
	return NewActuatorWithDeps(
		mgr,
		alicloudclient.NewClientFactory(),
		terraformer.DefaultFactory(),
		DefaultTerraformOps(),
		machineImageOwnerSecretRef,
		toBeSharedImageIDs,
		disableProjectedTokenMount,
	)
}

// NewActuatorWithDeps instantiates an actuator with the given dependencies.
func NewActuatorWithDeps(
	mgr manager.Manager,
	newClientFactory alicloudclient.ClientFactory,
	terraformerFactory terraformer.Factory,
	terraformChartOps TerraformChartOps,
	machineImageOwnerSecretRef *corev1.SecretReference,
	toBeSharedImageIDs []string,
	disableProjectedTokenMount bool,
) (infrastructure.Actuator, error) {
	a := &actuator{
		client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		restConfig: mgr.GetConfig(),
		decoder:    serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),

		newClientFactory:           newClientFactory,
		terraformerFactory:         terraformerFactory,
		terraformChartOps:          terraformChartOps,
		machineImageOwnerSecretRef: machineImageOwnerSecretRef,
		toBeSharedImageIDs:         toBeSharedImageIDs,
		disableProjectedTokenMount: disableProjectedTokenMount,
	}

	if a.machineImageOwnerSecretRef != nil {
		machineImageOwnerSecret := &corev1.Secret{}
		err := mgr.GetAPIReader().Get(context.Background(), client.ObjectKey{
			Name:      a.machineImageOwnerSecretRef.Name,
			Namespace: a.machineImageOwnerSecretRef.Namespace,
		}, machineImageOwnerSecret)
		if err != nil {
			return nil, err
		}
		seedCloudProviderCredentials, err := alicloud.ReadSecretCredentials(machineImageOwnerSecret, false)
		if err != nil {
			return nil, err
		}
		a.alicloudECSClient, err = a.newClientFactory.NewECSClient("", seedCloudProviderCredentials.AccessKeyID, seedCloudProviderCredentials.AccessKeySecret)
		return nil, err
	}

	return a, nil
}

type actuator struct {
	client     client.Client
	scheme     *runtime.Scheme
	decoder    runtime.Decoder
	restConfig *rest.Config

	alicloudECSClient  alicloudclient.ECS
	newClientFactory   alicloudclient.ClientFactory
	terraformerFactory terraformer.Factory
	terraformChartOps  TerraformChartOps

	machineImageOwnerSecretRef *corev1.SecretReference
	toBeSharedImageIDs         []string
	disableProjectedTokenMount bool
}

func (a *actuator) getConfigAndCredentialsForInfra(ctx context.Context, infra *extensionsv1alpha1.Infrastructure) (*alicloudv1alpha1.InfrastructureConfig, *alicloud.Credentials, error) {
	config := &alicloudv1alpha1.InfrastructureConfig{}
	if _, _, err := a.decoder.Decode(infra.Spec.ProviderConfig.Raw, nil, config); err != nil {
		return nil, nil, err
	}

	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, a.client, &infra.Spec.SecretRef)
	if err != nil {
		return nil, nil, err
	}

	return config, credentials, nil
}

func (a *actuator) convertImageListToV1alpha1(machineImages []apisalicloud.MachineImage) ([]alicloudv1alpha1.MachineImage, error) {
	var result []alicloudv1alpha1.MachineImage
	for _, image := range machineImages {
		converted := &alicloudv1alpha1.MachineImage{}
		if err := a.scheme.Convert(&image, converted, nil); err != nil {
			return nil, err
		}

		result = append(result, *converted)
	}

	return result, nil
}

// ensureOldSSHKeyDetached ensures the compatibility when ssh key is generated in the current cluster via terraform.
func (a *actuator) ensureOldSSHKeyDetached(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure) error {
	if infra.Status.ProviderStatus == nil {
		return nil
	}

	infrastructureStatus := &alicloudv1alpha1.InfrastructureStatus{}
	if _, _, err := a.decoder.Decode(infra.Status.ProviderStatus.Raw, nil, infrastructureStatus); err != nil {
		return err
	}

	// nolint
	if infrastructureStatus.KeyPairName == "" {
		return nil
	}

	_, shootCloudProviderCredentials, err := a.getConfigAndCredentialsForInfra(ctx, infra)
	if err != nil {
		return err
	}

	shootAlicloudECSClient, err := a.newClientFactory.NewECSClient(infra.Spec.Region, shootCloudProviderCredentials.AccessKeyID, shootCloudProviderCredentials.AccessKeySecret)
	if err != nil {
		return err
	}

	// nolint
	log.V(2).Info("Detaching ssh key pair from ECS instances", "keypair", infrastructureStatus.KeyPairName)
	// nolint
	err = shootAlicloudECSClient.DetachECSInstancesFromSSHKeyPair(infrastructureStatus.KeyPairName)
	// nolint
	log.V(2).Info("Finished detaching ssh key pair from ECS instances", "keypair", infrastructureStatus.KeyPairName)

	return err
}

func (a *actuator) cleanupServiceLoadBalancers(ctx context.Context, infra *extensionsv1alpha1.Infrastructure) error {
	_, shootCloudProviderCredentials, err := a.getConfigAndCredentialsForInfra(ctx, infra)
	if err != nil {
		return err
	}

	shootAlicloudSLBClient, err := a.newClientFactory.NewSLBClient(infra.Spec.Region, shootCloudProviderCredentials.AccessKeyID, shootCloudProviderCredentials.AccessKeySecret)
	if err != nil {
		return err
	}

	loadBalancerIDs, err := shootAlicloudSLBClient.GetLoadBalancerIDs(ctx, infra.Spec.Region)
	if err != nil {
		return err
	}
	// SLBs created by Alicloud CCM do not have association with VPCs, so can only be iterated to check
	// if one SLB is related to this specific Shoot.
	for _, loadBalancerID := range loadBalancerIDs {
		vServerGroupName, err := shootAlicloudSLBClient.GetFirstVServerGroupName(ctx, infra.Spec.Region, loadBalancerID)
		if err != nil {
			return err
		}
		if vServerGroupName == "" {
			continue
		}

		// Get the last slice of VServerGroupName string divided by '/' which is the clusterid.
		slices := strings.Split(vServerGroupName, "/")
		clusterID := slices[len(slices)-1]
		if clusterID == infra.Namespace {
			err = shootAlicloudSLBClient.SetLoadBalancerDeleteProtection(ctx, infra.Spec.Region, loadBalancerID, false)
			if err != nil {
				return err
			}
			err = shootAlicloudSLBClient.DeleteLoadBalancer(ctx, infra.Spec.Region, loadBalancerID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *actuator) cleanupTerraformerResources(ctx context.Context, log logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure) error {
	tf, err := common.NewTerraformer(log, a.terraformerFactory, a.restConfig, TerraformerPurpose, infrastructure, a.disableProjectedTokenMount)
	if err != nil {
		return fmt.Errorf("could not create terraformer object: %w", err)
	}
	if err := tf.EnsureCleanedUp(ctx); err != nil {
		return err
	}
	if err := tf.CleanupConfiguration(ctx); err != nil {
		return err
	}
	return tf.RemoveTerraformerFinalizerFromConfig(ctx)
}

// ensureServiceLinkedRole is to check if service linked role exists, if not create one.
func (a *actuator) ensureServiceLinkedRole(_ context.Context, infra *extensionsv1alpha1.Infrastructure, credentials *alicloud.Credentials) error {
	client, err := a.newClientFactory.NewRAMClient(infra.Spec.Region, credentials.AccessKeyID, credentials.AccessKeySecret)
	if err != nil {
		return err
	}

	serviceLinkedRole, err := client.GetServiceLinkedRole(alicloud.ServiceLinkedRoleForNATGateway)
	if err != nil {
		return err
	}

	if serviceLinkedRole == nil {
		if err := client.CreateServiceLinkedRole(infra.Spec.Region, alicloud.ServiceForNATGateway); err != nil {
			return err
		}
	}

	return nil
}
