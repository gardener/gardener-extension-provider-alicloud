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
	extensionspredicate "github.com/gardener/gardener/extensions/pkg/predicate"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	corev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
)

var logger = log.Log.WithName("alicloud-mutator-webhook")

// NewShootsWebhook creates a new mutation webhook for shoots.
func NewShootsWebhook(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	virtualGardenclient := mgr.GetClient()
	apiReader := mgr.GetAPIReader()
	scheme := mgr.GetScheme()
	decoder := serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
	return extensionswebhook.New(mgr, extensionswebhook.Args{
		Provider:   alicloud.Type,
		Name:       ShootMutatorName,
		Path:       MutatorPath + "/shoots",
		Predicates: []predicate.Predicate{extensionspredicate.GardenCoreProviderType(alicloud.Type)},
		Mutators: map[extensionswebhook.Mutator][]client.Object{
			NewShootMutator(virtualGardenclient, apiReader, decoder): {&corev1beta1.Shoot{}},
		},
	})
}
