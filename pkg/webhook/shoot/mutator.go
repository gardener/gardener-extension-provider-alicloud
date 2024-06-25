// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
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
func (m *mutator) Mutate(_ context.Context, new, old client.Object) error {
	svc, ok := new.(*corev1.Service)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	if svc.GetDeletionTimestamp() != nil {
		return nil
	}

	if svc.GetName() == "addons-nginx-ingress-controller" {
		var oldSvc *corev1.Service
		if old != nil {
			var ok bool
			oldSvc, ok = old.(*corev1.Service)
			if !ok {
				return fmt.Errorf("wrong object type %T", old)
			}
		}

		extensionswebhook.LogMutation(logger, svc.Kind, svc.Namespace, svc.Name)
		webhookutils.MutateAnnotation(svc, oldSvc, m.service.BackendLoadBalancerSpec)
		webhookutils.MutateExternalTrafficPolicy(svc, oldSvc)
	}

	return nil
}
