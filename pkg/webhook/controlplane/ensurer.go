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
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/coreos/go-systemd/v22/unit"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/gardener/gardener/pkg/utils/version"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/utils/pointer"
)

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

// EnsureKubeAPIServerDeployment ensures that the kube-apiserver deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerDeployment(ctx context.Context, gctx gcontext.GardenContext, new, old *appsv1.Deployment) error {
	ps := &new.Spec.Template.Spec
	if c := extensionswebhook.ContainerWithName(ps.Containers, "kube-apiserver"); c != nil {
		cluster, err := gctx.GetCluster(ctx)
		if err != nil {
			return err
		}
		ver, err := semver.NewVersion(cluster.Shoot.Spec.Kubernetes.Version)
		if err != nil {
			return fmt.Errorf("cannot parse shoot k8s cluster version: %v", err)
		}
		ensureKubeAPIServerCommandLineArgs(c, ver)
	}
	return nil
}

// EnsureKubeControllerManagerDeployment ensures that the kube-controller-manager deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeControllerManagerDeployment(ctx context.Context, gctx gcontext.GardenContext, new, old *appsv1.Deployment) error {
	ps := &new.Spec.Template.Spec
	if c := extensionswebhook.ContainerWithName(ps.Containers, "kube-controller-manager"); c != nil {
		ensureKubeControllerManagerCommandLineArgs(c)
	}
	return nil
}

func ensureKubeAPIServerCommandLineArgs(c *corev1.Container, ver *semver.Version) {
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
func (e *ensurer) EnsureKubeletServiceUnitOptions(ctx context.Context, gctx gcontext.GardenContext, kubeletVersion *semver.Version, new, old []*unit.UnitOption) ([]*unit.UnitOption, error) {
	if opt := extensionswebhook.UnitOptionWithSectionAndName(new, "Service", "ExecStart"); opt != nil {
		command := extensionswebhook.DeserializeCommandLine(opt.Value)
		command = ensureKubeletCommandLineArgs(command, kubeletVersion)
		opt.Value = extensionswebhook.SerializeCommandLine(command, 1, " \\\n    ")
	}
	new = extensionswebhook.EnsureUnitOption(new, &unit.UnitOption{
		Section: "Service",
		Name:    "ExecStartPre",
		Value:   getValueOfKubeletPreStart(kubeletVersion),
	})
	return new, nil
}
func getValueOfKubeletPreStart(kubeletVersion *semver.Version) string {
	if version.ConstraintK8sGreaterEqual123.Check(kubeletVersion) {
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
	} else {
		/*
		 * # Set environment PROVIDER_ID to /var/lib/kubelet/extra_args
		 *
		 * grep -sq PROVIDER_ID /var/lib/kubelet/extra_args
		 * if [ $? -ne 0 ]; then
		 *   META_EP=http://100.100.100.200/latest/meta-data
		 *   PROVIDER_ID=`wget -qO- $META_EP/region-id`.`wget -qO- $META_EP/instance-id`
		 *   echo PROVIDER_ID=$PROVIDER_ID >> /var/lib/kubelet/extra_args
		 *   echo PROVIDER_ID=$PROVIDER_ID has been written to /var/lib/kubelet/extra_args
		 * fi
		 */
		//This doesn't work: /bin/sh -c "$(echo  Z3JlcCAtc3EgUFJPVklERVJfSUQgL2V0Yy9lbnZpcm9ubWVudAppZiBbICQ.... | base64 -d)"
		return `/bin/sh -c "echo Z3JlcCAtc3EgUFJPVklERVJfSUQgL3Zhci9saWIva3ViZWxldC9leHRyYV9hcmdzCmlmIFsgJD8gLW5lIDAgXTsgdGhlbgpNRVRBX0VQPWh0dHA6Ly8xMDAuMTAwLjEwMC4yMDAvbGF0ZXN0L21ldGEtZGF0YQpQUk9WSURFUl9JRD1gd2dldCAtcU8tICRNRVRBX0VQL3JlZ2lvbi1pZGAuYHdnZXQgLXFPLSAkTUVUQV9FUC9pbnN0YW5jZS1pZGAKZWNobyBQUk9WSURFUl9JRD0kUFJPVklERVJfSUQgPj4gL3Zhci9saWIva3ViZWxldC9leHRyYV9hcmdzCmVjaG8gUFJPVklERVJfSUQ9JFBST1ZJREVSX0lEIGhhcyBiZWVuIHdyaXR0ZW4gdG8gL3Zhci9saWIva3ViZWxldC9leHRyYV9hcmdzCmZpCg==| base64 -d > /var/lib/kubelet/gardener-set-provider-id && chmod +x /var/lib/kubelet/gardener-set-provider-id && /var/lib/kubelet/gardener-set-provider-id"`
	}
}
func ensureKubeletCommandLineArgs(command []string, kubeletVersion *semver.Version) []string {
	// TODO: Figure out how to provide the provider-id via the kubelet config file (as of Kubernetes 1.19 the kubelet config
	// offers a new `providerID` field which can be used, and it's expected that `--provider-id` will be deprecated eventually).
	// Today, the problem is that the provider ID is determined dynamically using the script above, but the kubelet config cannot
	// reference environment variables like it's possible today with the CLI parameters.
	// See https://github.com/kubernetes/kubernetes/pull/90494
	command = extensionswebhook.EnsureStringWithPrefix(command, "--cloud-provider=", "external")

	if !version.ConstraintK8sGreaterEqual123.Check(kubeletVersion) {
		command = extensionswebhook.EnsureStringWithPrefix(command, "--provider-id=", "${PROVIDER_ID}")
		command = extensionswebhook.EnsureStringWithPrefix(command, "--enable-controller-attach-detach=", "true")
	}

	return command
}

// EnsureKubeletConfiguration ensures that the kubelet configuration conforms to the provider requirements.
func (e *ensurer) EnsureKubeletConfiguration(ctx context.Context, gctx gcontext.GardenContext, kubeletVersion *semver.Version, new, old *kubeletconfigv1beta1.KubeletConfiguration) error {
	if version.ConstraintK8sGreaterEqual123.Check(kubeletVersion) {
		new.EnableControllerAttachDetach = pointer.Bool(true)
		new.ProviderID = "place_holder_of_providerid"
	}

	return nil
}
