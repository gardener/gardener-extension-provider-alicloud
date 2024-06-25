// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/config"
	webhookutils "github.com/gardener/gardener-extension-provider-alicloud/pkg/webhook/utils"
)

type mutator struct {
	logger  logr.Logger
	service *config.Service
}

// NewMutator creates a new Mutator that mutates resources in the shoot cluster.
func NewMutator(service *config.Service) extensionswebhook.Mutator {
	return &mutator{
		logger:  log.Log.WithName("shoot-mutator"),
		service: service,
	}
}

// Mutate mutates resources.
func (m *mutator) Mutate(ctx context.Context, new, old client.Object) error {
	acc, err := meta.Accessor(new)
	if err != nil {
		return fmt.Errorf("could not create accessor during webhook: %w", err)
	}
	// If the object does have a deletion timestamp then we don't want to mutate anything.
	if acc.GetDeletionTimestamp() != nil {
		return nil
	}

	switch x := new.(type) {
	case *corev1.Service:
		if x.Name == "addons-nginx-ingress-controller" {
			var oldSvc *corev1.Service
			if old != nil {
				var ok bool
				oldSvc, ok = old.(*corev1.Service)
				if !ok {
					return fmt.Errorf("could not cast old object to corev1.Service: %w", err)
				}
			}

			extensionswebhook.LogMutation(logger, x.Kind, x.Namespace, x.Name)
			webhookutils.MutateAnnotation(x, oldSvc, m.service.BackendLoadBalancerSpec)
			webhookutils.MutateExternalTrafficPolicy(x, oldSvc)
		}
	case *appsv1.Deployment:
		if x.Name == "metrics-server" {
			extensionswebhook.LogMutation(logger, x.Kind, x.Namespace, x.Name)
			m.mutateMetricsServerDeployment(ctx, x)
		}
	}
	return nil
}

func (m *mutator) mutateMetricsServerDeployment(_ context.Context, dep *appsv1.Deployment) {
	ps := &dep.Spec.Template.Spec
	if c := extensionswebhook.ContainerWithName(ps.Containers, "metrics-server"); c != nil {
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--kubelet-preferred-address-types=", "InternalIP")
	}
}
