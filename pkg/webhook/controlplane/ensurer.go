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

	"github.com/Masterminds/semver"
	"github.com/coreos/go-systemd/v22/unit"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/component/machinecontrollermanager"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/utils/pointer"

	"github.com/gardener/gardener-extension-provider-alicloud/imagevector"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
)

// NewEnsurer creates a new controlplane ensurer.
func NewEnsurer(logger logr.Logger, gardenletManagesMCM bool) genericmutator.Ensurer {
	return &ensurer{
		logger:              logger.WithName("alicloud-controlplane-ensurer"),
		gardenletManagesMCM: gardenletManagesMCM,
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	logger              logr.Logger
	gardenletManagesMCM bool
}

// ImageVector is exposed for testing.
var ImageVector = imagevector.ImageVector()

// EnsureMachineControllerManagerDeployment ensures that the machine-controller-manager deployment conforms to the provider requirements.
func (e *ensurer) EnsureMachineControllerManagerDeployment(_ context.Context, _ gcontext.GardenContext, newObj, _ *appsv1.Deployment) error {
	if !e.gardenletManagesMCM {
		return nil
	}

	image, err := ImageVector.FindImage(alicloud.MachineControllerManagerProviderAlicloudImageName)
	if err != nil {
		return err
	}

	newObj.Spec.Template.Spec.Containers = extensionswebhook.EnsureContainerWithName(
		newObj.Spec.Template.Spec.Containers,
		machinecontrollermanager.ProviderSidecarContainer(newObj.Namespace, alicloud.Name, image.String()),
	)
	return nil
}

// EnsureMachineControllerManagerVPA ensures that the machine-controller-manager VPA conforms to the provider requirements.
func (e *ensurer) EnsureMachineControllerManagerVPA(_ context.Context, _ gcontext.GardenContext, newObj, _ *vpaautoscalingv1.VerticalPodAutoscaler) error {
	if !e.gardenletManagesMCM {
		return nil
	}

	var (
		minAllowed = corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		}
		maxAllowed = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("5G"),
		}
	)

	if newObj.Spec.ResourcePolicy == nil {
		newObj.Spec.ResourcePolicy = &vpaautoscalingv1.PodResourcePolicy{}
	}

	newObj.Spec.ResourcePolicy.ContainerPolicies = extensionswebhook.EnsureVPAContainerResourcePolicyWithName(
		newObj.Spec.ResourcePolicy.ContainerPolicies,
		machinecontrollermanager.ProviderSidecarVPAContainerPolicy(alicloud.Name, minAllowed, maxAllowed),
	)
	return nil
}

// EnsureKubeAPIServerDeployment ensures that the kube-apiserver deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerDeployment(_ context.Context, _ gcontext.GardenContext, newObj, _ *appsv1.Deployment) error {
	ps := &newObj.Spec.Template.Spec

	// TODO: This label approach is deprecated and no longer needed in the future. Remove it as soon as gardener/gardener@v1.75 has been released.
	metav1.SetMetaDataLabel(&newObj.Spec.Template.ObjectMeta, gutil.NetworkPolicyLabel(alicloud.CSISnapshotValidationName, 443), v1beta1constants.LabelNetworkPolicyAllowed)

	if c := extensionswebhook.ContainerWithName(ps.Containers, "kube-apiserver"); c != nil {
		ensureKubeAPIServerCommandLineArgs(c)
	}
	return nil
}

// EnsureKubeControllerManagerDeployment ensures that the kube-controller-manager deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeControllerManagerDeployment(_ context.Context, _ gcontext.GardenContext, newObj, _ *appsv1.Deployment) error {
	ps := &newObj.Spec.Template.Spec
	if c := extensionswebhook.ContainerWithName(ps.Containers, "kube-controller-manager"); c != nil {
		ensureKubeControllerManagerCommandLineArgs(c)
	}
	return nil
}

func ensureKubeAPIServerCommandLineArgs(c *corev1.Container) {
	// Ensure CSI-related admission plugins
	c.Command = extensionswebhook.EnsureNoStringWithPrefixContains(c.Command, "--enable-admission-plugins=",
		"PersistentVolumeLabel", ",")
	c.Command = extensionswebhook.EnsureStringWithPrefixContains(c.Command, "--disable-admission-plugins=",
		"PersistentVolumeLabel", ",")

	// Ensure CSI-related feature gates
	c.Command = extensionswebhook.EnsureNoStringWithPrefixContains(c.Command, "--feature-gates=",
		"ExpandInUsePersistentVolumes=false", ",")
	c.Command = extensionswebhook.EnsureNoStringWithPrefixContains(c.Command, "--feature-gates=",
		"ExpandCSIVolumes=false", ",")
	c.Command = extensionswebhook.EnsureNoStringWithPrefixContains(c.Command, "--feature-gates=",
		"CSINodeInfo=false", ",")
	c.Command = extensionswebhook.EnsureNoStringWithPrefixContains(c.Command, "--feature-gates=",
		"CSIDriverRegistry=false", ",")
}

func ensureKubeControllerManagerCommandLineArgs(c *corev1.Container) {
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--cloud-provider=", "external")
}

// EnsureKubeletServiceUnitOptions ensures that the kubelet.service unit options conform to the provider requirements.
func (e *ensurer) EnsureKubeletServiceUnitOptions(_ context.Context, _ gcontext.GardenContext, _ *semver.Version, newObj, _ []*unit.UnitOption) ([]*unit.UnitOption, error) {
	if opt := extensionswebhook.UnitOptionWithSectionAndName(newObj, "Service", "ExecStart"); opt != nil {
		command := extensionswebhook.DeserializeCommandLine(opt.Value)
		command = ensureKubeletCommandLineArgs(command)
		opt.Value = extensionswebhook.SerializeCommandLine(command, 1, " \\\n    ")
	}
	newObj = extensionswebhook.EnsureUnitOption(newObj, &unit.UnitOption{
		Section: "Service",
		Name:    "ExecStartPre",
		Value:   getValueOfKubeletPreStart(),
	})
	return newObj, nil
}

func getValueOfKubeletPreStart() string {
	/*
		# set providerid in /var/lib/kubelet/config/kubelet
		grep -sq place_holder_of_providerid /var/lib/kubelet/config/kubelet
		if [ $? -eq 0 ]; then
		    META_EP=http://100.100.100.200/latest/meta-data
		    PROVIDER_ID=`wget -qO- $META_EP/region-id`.`wget -qO- $META_EP/instance-id`
		    sudo sed  -i "s/place_holder_of_providerid/${PROVIDER_ID}/g" /var/lib/kubelet/config/kubelet
		    echo "providerID= $PROVIDER_ID has been written to /var/lib/kubelet/config/kubelet"
		fi
	*/
	return `/bin/sh -c "echo IyBzZXQgcHJvdmlkZXJpZCBpbiAvdmFyL2xpYi9rdWJlbGV0L2NvbmZpZy9rdWJlbGV0CmdyZXAgLXNxIHBsYWNlX2hvbGRlcl9vZl9wcm92aWRlcmlkIC92YXIvbGliL2t1YmVsZXQvY29uZmlnL2t1YmVsZXQKaWYgWyAkPyAtZXEgMCBdOyB0aGVuCiAgICBNRVRBX0VQPWh0dHA6Ly8xMDAuMTAwLjEwMC4yMDAvbGF0ZXN0L21ldGEtZGF0YQogICAgUFJPVklERVJfSUQ9YHdnZXQgLXFPLSAkTUVUQV9FUC9yZWdpb24taWRgLmB3Z2V0IC1xTy0gJE1FVEFfRVAvaW5zdGFuY2UtaWRgCiAgICBzdWRvIHNlZCAgLWkgInMvcGxhY2VfaG9sZGVyX29mX3Byb3ZpZGVyaWQvJHtQUk9WSURFUl9JRH0vZyIgL3Zhci9saWIva3ViZWxldC9jb25maWcva3ViZWxldAogICAgZWNobyAicHJvdmlkZXJJRD0gJFBST1ZJREVSX0lEIGhhcyBiZWVuIHdyaXR0ZW4gdG8gL3Zhci9saWIva3ViZWxldC9jb25maWcva3ViZWxldCIKZmkK| base64 -d > /var/lib/kubelet/gardener-set-provider-id && chmod +x /var/lib/kubelet/gardener-set-provider-id && /var/lib/kubelet/gardener-set-provider-id"`
}

func ensureKubeletCommandLineArgs(command []string) []string {
	// TODO: Figure out how to provide the provider-id via the kubelet config file (as of Kubernetes 1.19 the kubelet config
	// offers a new `providerID` field which can be used, and it's expected that `--provider-id` will be deprecated eventually).
	// Today, the problem is that the provider ID is determined dynamically using the script above, but the kubelet config cannot
	// reference environment variables like it's possible today with the CLI parameters.
	// See https://github.com/kubernetes/kubernetes/pull/90494
	return extensionswebhook.EnsureStringWithPrefix(command, "--cloud-provider=", "external")
}

// EnsureKubeletConfiguration ensures that the kubelet configuration conforms to the provider requirements.
func (e *ensurer) EnsureKubeletConfiguration(_ context.Context, _ gcontext.GardenContext, _ *semver.Version, newObj, _ *kubeletconfigv1beta1.KubeletConfiguration) error {
	newObj.EnableControllerAttachDetach = pointer.Bool(true)
	newObj.ProviderID = "place_holder_of_providerid"

	return nil
}
