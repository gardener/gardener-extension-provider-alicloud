// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:generate mockgen -package=client -destination=mocks.go github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client ClientFactory,ECS,STS,SLB,VPC,OSS,RAM,ROS,NLB

package client
