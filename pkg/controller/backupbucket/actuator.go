// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package backupbucket

import (
	"github.com/gardener/gardener/extensions/pkg/controller/backupbucket"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
)

type actuator struct {
	backupbucket.Actuator
	client           client.Client
	aliClientFactory alicloudclient.ClientFactory
	action           Action
}

// Action is interface which defines action.
type Action interface {
	// Do performs an action.
	Do() error
}

// ActionFunc is a function that implements Action.
type ActionFunc func() error

// Do performs an action.
func (f ActionFunc) Do() error {
	return f()
}

// NewActuator creates a new Actuator that creates/updates backup-bucket.
func NewActuator(mgr manager.Manager, aliClientFactory alicloudclient.ClientFactory) backupbucket.Actuator {
	return &actuator{
		client:           mgr.GetClient(),
		aliClientFactory: aliClientFactory,
	}
}
