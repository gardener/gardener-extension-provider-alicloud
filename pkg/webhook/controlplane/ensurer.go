// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"

	"github.com/Masterminds/semver/v3"
	"github.com/coreos/go-systemd/v22/unit"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/component/nodemanagement/machinecontrollermanager"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-provider-alicloud/imagevector"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
)

var (
	// constraintK8sLess131 is a version constraint for versions < 1.31.
	//
	// TODO(ialidzhikov): Replace with versionutils.ConstraintK8sLess131 when vendoring a gardener/gardener version
	// that contains https://github.com/gardener/gardener/pull/10472.
	constraintK8sLess131 *semver.Constraints
)

func init() {
	var err error
	constraintK8sLess131, err = semver.NewConstraint("< 1.31-0")
	utilruntime.Must(err)
}

// NewEnsurer creates a new controlplane ensurer.
func NewEnsurer(logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		logger: logger.WithName("alicloud-controlplane-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	logger logr.Logger
}

// ImageVector is exposed for testing.
var ImageVector = imagevector.ImageVector()

// EnsureMachineControllerManagerDeployment ensures that the machine-controller-manager deployment conforms to the provider requirements.
func (e *ensurer) EnsureMachineControllerManagerDeployment(_ context.Context, _ gcontext.GardenContext, newObj, _ *appsv1.Deployment) error {
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
	if newObj.Spec.ResourcePolicy == nil {
		newObj.Spec.ResourcePolicy = &vpaautoscalingv1.PodResourcePolicy{}
	}

	newObj.Spec.ResourcePolicy.ContainerPolicies = extensionswebhook.EnsureVPAContainerResourcePolicyWithName(
		newObj.Spec.ResourcePolicy.ContainerPolicies,
		machinecontrollermanager.ProviderSidecarVPAContainerPolicy(alicloud.Name),
	)
	return nil
}

// EnsureKubeAPIServerDeployment ensures that the kube-apiserver deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerDeployment(ctx context.Context, gctx gcontext.GardenContext, newObj, _ *appsv1.Deployment) error {
	ps := &newObj.Spec.Template.Spec

	// TODO: This label approach is deprecated and no longer needed in the future. Remove it as soon as gardener/gardener@v1.75 has been released.
	metav1.SetMetaDataLabel(&newObj.Spec.Template.ObjectMeta, gutil.NetworkPolicyLabel(alicloud.CSISnapshotValidationName, 443), v1beta1constants.LabelNetworkPolicyAllowed)

	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return err
	}

	k8sVersion, err := semver.NewVersion(cluster.Shoot.Spec.Kubernetes.Version)
	if err != nil {
		return err
	}

	if c := extensionswebhook.ContainerWithName(ps.Containers, "kube-apiserver"); c != nil {
		ensureKubeAPIServerCommandLineArgs(c, k8sVersion)
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

func ensureKubeAPIServerCommandLineArgs(c *corev1.Container, k8sVersion *semver.Version) {
	// Ensure CSI-related admission plugins
	if constraintK8sLess131.Check(k8sVersion) {
		c.Command = extensionswebhook.EnsureNoStringWithPrefixContains(c.Command, "--enable-admission-plugins=",
			"PersistentVolumeLabel", ",")
		c.Command = extensionswebhook.EnsureStringWithPrefixContains(c.Command, "--disable-admission-plugins=",
			"PersistentVolumeLabel", ",")
	}

	// TODO: Remove the below lines when K8s < 1.27 is no longer supported.
	// The ExpandCSIVolumes and ExpandInUsePersistentVolumes feature gates are GA since 1.24,
	// however they are not locked to default (true). They are removed in K8s 1.27.
	c.Command = extensionswebhook.EnsureNoStringWithPrefixContains(c.Command, "--feature-gates=",
		"ExpandInUsePersistentVolumes=false", ",")
	c.Command = extensionswebhook.EnsureNoStringWithPrefixContains(c.Command, "--feature-gates=",
		"ExpandCSIVolumes=false", ",")
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
	newObj.EnableControllerAttachDetach = ptr.To(true)
	newObj.ProviderID = "place_holder_of_providerid"

	return nil
}
