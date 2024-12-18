// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package mutator

import (
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	corev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
)

var logger = log.Log.WithName("alicloud-mutator-webhook")

// NewShootsWebhook creates a new mutation webhook for shoots.
func NewShootsWebhook(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	return extensionswebhook.New(mgr, extensionswebhook.Args{
		Provider: alicloud.Type,
		Name:     ShootMutatorName,
		Path:     MutatorPath + "/shoots",
		Mutators: map[extensionswebhook.Mutator][]extensionswebhook.Type{
			NewShootMutator(mgr): {{Obj: &corev1beta1.Shoot{}}},
		},
		Target: extensionswebhook.TargetSeed,
		ObjectSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"provider.extensions.gardener.cloud/alicloud": "true"},
		},
	})
}
