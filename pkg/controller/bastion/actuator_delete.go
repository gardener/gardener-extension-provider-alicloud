// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package bastion

import (
	"context"
	"fmt"
	"time"

	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/util"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	aliclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
)

func (a *actuator) Delete(ctx context.Context, log logr.Logger, bastion *extensionsv1alpha1.Bastion, cluster *controller.Cluster) error {
	log.Info("Bastion deletion operation")
	opt, err := DetermineOptions(bastion, cluster)
	if err != nil {
		return err
	}

	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, a.client, &opt.SecretReference)
	if err != nil {
		return err
	}

	aliCloudECSClient, err := a.newClientFactory.NewECSClient(opt.Region, credentials.AccessKeyID, credentials.AccessKeySecret)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	err = removeBastionInstance(aliCloudECSClient, opt)
	if err != nil {
		return util.DetermineError(fmt.Errorf("failed to terminate bastion instance: %w", err), helper.KnownCodes)
	}

	log.Info("Instance remove processing", "instance", opt.BastionInstanceName)

	time.Sleep(10 * time.Second)

	err = removeSecurityGroup(aliCloudECSClient, opt)
	if err != nil {
		return util.DetermineError(fmt.Errorf("failed to remove security group: %w", err), helper.KnownCodes)
	}

	log.Info("security group removed:", "security group", opt.SecurityGroupName)
	return nil
}

func (a *actuator) ForceDelete(_ context.Context, _ logr.Logger, _ *extensionsv1alpha1.Bastion, _ *controller.Cluster) error {
	return nil
}

func removeSecurityGroup(c aliclient.ECS, opt *Options) error {
	response, err := c.GetSecurityGroup(opt.SecurityGroupName)
	if err != nil {
		return err
	}

	if len(response.SecurityGroups.SecurityGroup) == 0 {
		return nil
	}

	return c.DeleteSecurityGroups(response.SecurityGroups.SecurityGroup[0].SecurityGroupId)
}

func removeBastionInstance(c aliclient.ECS, opt *Options) error {
	response, err := c.GetInstances(opt.BastionInstanceName)
	if err != nil {
		return err
	}

	if len(response.Instances.Instance) == 0 {
		return nil
	}

	return c.DeleteInstances(response.Instances.Instance[0].InstanceId, true)
}
