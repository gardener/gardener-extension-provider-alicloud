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

package controlplaneexposure

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/config"
	webhookutils "github.com/gardener/gardener-extension-provider-alicloud/pkg/webhook/utils"
	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"

	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewEnsurer creates a new controlplaneexposure ensurer.
func NewEnsurer(etcdStorage *config.ETCDStorage, kubeAPIServer *config.KubeAPIServer, logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		etcdStorage:   etcdStorage,
		kubeAPIServer: kubeAPIServer,
		logger:        logger.WithName("alicloud-controlplaneexposure-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	etcdStorage   *config.ETCDStorage
	kubeAPIServer *config.KubeAPIServer
	client        client.Client
	logger        logr.Logger
}

// InjectClient injects the given client into the ensurer.
func (e *ensurer) InjectClient(client client.Client) error {
	e.client = client
	return nil
}

// EnsureKubeAPIServerService ensures that the kube-apiserver service conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerService(ctx context.Context, ectx genericmutator.EnsurerContext, new, old *corev1.Service) error {
	if e.kubeAPIServer.MutateExternalTrafficPolicy {
		return webhookutils.MutateLBService(new, old)
	}

	return nil
}

// EnsureKubeAPIServerDeployment ensures that the kube-apiserver deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerDeployment(ctx context.Context, ectx genericmutator.EnsurerContext, new, old *appsv1.Deployment) error {
	if v1beta1helper.IsAPIServerExposureManaged(new) {
		return nil
	}

	cluster, err := controller.GetCluster(ctx, e.client, new.Namespace)
	if err != nil {
		return err
	}

	if controller.IsHibernated(cluster) {
		return nil
	}

	// Get load balancer address of the kube-apiserver service
	address, err := kutil.GetLoadBalancerIngress(ctx, e.client, new.Namespace, v1beta1constants.DeploymentNameKubeAPIServer)
	if err != nil {
		return errors.Wrap(err, "could not get kube-apiserver service load balancer address")
	}

	if c := extensionswebhook.ContainerWithName(new.Spec.Template.Spec.Containers, "kube-apiserver"); c != nil {
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--advertise-address=", address)
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--external-hostname=", address)
	}
	return nil
}

// EnsureETCD ensures that the etcd conform to the provider requirements.
func (e *ensurer) EnsureETCD(ctx context.Context, ectx genericmutator.EnsurerContext, new, old *druidv1alpha1.Etcd) error {
	capacity := resource.MustParse("20Gi")
	class := ""

	if new.Name == v1beta1constants.ETCDMain && e.etcdStorage != nil {
		if e.etcdStorage.Capacity != nil {
			capacity = *e.etcdStorage.Capacity
		}
		if e.etcdStorage.ClassName != nil {
			class = *e.etcdStorage.ClassName
		}
	}

	new.Spec.StorageClass = &class
	new.Spec.StorageCapacity = &capacity

	return nil
}
