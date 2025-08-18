// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package bastion

import (
	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
)

const (
	// sshPort is the default SSH Port used for bastion ingress firewall rule
	sshPort = "22"
)

type actuator struct {
	client           client.Client
	newClientFactory alicloudclient.ClientFactory
}

func newActuator(mgr manager.Manager) bastion.Actuator {
	return &actuator{
		client:           mgr.GetClient(),
		newClientFactory: alicloudclient.NewClientFactory(),
	}
}
