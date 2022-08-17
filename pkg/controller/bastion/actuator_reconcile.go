// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package bastion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	aliclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	alicloudapi "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	ctrlerror "github.com/gardener/gardener/pkg/controllerutils/reconciler"
	"github.com/gardener/gardener/pkg/extensions"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// bastionEndpoints holds the endpoints the bastion host provides
type bastionEndpoints struct {
	// private is the private endpoint of the bastion. It is required when opening a port on the worker node to allow SSH access from the bastion
	private *corev1.LoadBalancerIngress
	//  public is the public endpoint where the end user connects to establish the SSH connection.
	public *corev1.LoadBalancerIngress
}

// Ready returns true if both public and private interfaces each have either
// an IP or a hostname or both.
func (be *bastionEndpoints) Ready() bool {
	return be != nil && IngressReady(be.private) && IngressReady(be.public)
}

func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, bastion *extensionsv1alpha1.Bastion, cluster *controller.Cluster) error {
	log.Info("Bastion reconciles operation")
	opt, err := DetermineOptions(bastion, cluster)
	if err != nil {
		return err
	}

	infrastructureStatus, err := getInfrastructureStatus(ctx, a, cluster)
	if err != nil {
		return err
	}

	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, a.client, &opt.SecretReference)
	if err != nil {
		return err
	}

	aliCloudECSClient, err := a.newClientFactory.NewECSClient(opt.Region, credentials.AccessKeyID, credentials.AccessKeySecret)
	if err != nil {
		return err
	}

	imageID := infrastructureStatus.MachineImages[0].ID
	vSwitchesZoneID := infrastructureStatus.VPC.VSwitches[0].Zone
	vSwitchesID := infrastructureStatus.VPC.VSwitches[0].ID
	vpcId := infrastructureStatus.VPC.ID
	shootSecurityGroupId := infrastructureStatus.VPC.SecurityGroups[0].ID

	var instanceTypeId string
	for cores := 1; cores <= 2; cores++ {
		instanceType, err := aliCloudECSClient.GetInstanceType(cores, vSwitchesZoneID)
		if err != nil {
			return err
		}

		if instanceType == nil ||
			len(instanceType.AvailableZones.AvailableZone) == 0 ||
			len(instanceType.AvailableZones.AvailableZone[0].AvailableResources.AvailableResource) == 0 ||
			len(instanceType.AvailableZones.AvailableZone[0].AvailableResources.AvailableResource[0].SupportedResources.SupportedResource) == 0 ||
			instanceType.AvailableZones.AvailableZone[0].AvailableResources.AvailableResource[0].SupportedResources.SupportedResource[0].Status != "Available" {
			continue
		}

		instanceTypeId = instanceType.AvailableZones.AvailableZone[0].AvailableResources.AvailableResource[0].SupportedResources.SupportedResource[0].Value
		break
	}

	if instanceTypeId == "" {
		if len(cluster.CloudProfile.Spec.MachineTypes) == 0 {
			return errors.New("failed to determine instanceTypeId from cloud profile as fallback. Machine types missing from cloud profile")
		}

		instanceTypeId = cluster.CloudProfile.Spec.MachineTypes[0].Name
		log.Info("falling back to first machine type of cloud profile as bastion instance type id", "instance type", cluster.CloudProfile.Spec.MachineTypes[0].Name)
	}

	securityGroupID, err := ensureSecurityGroup(aliCloudECSClient, opt.SecurityGroupName, vpcId, log)
	if err != nil {
		return err
	}

	instanceID, err := ensureComputeInstance(aliCloudECSClient, log, opt, securityGroupID, imageID, vSwitchesID, vSwitchesZoneID, instanceTypeId)
	if err != nil {
		return err
	}

	ready, err := isInstanceReady(aliCloudECSClient, opt)
	if err != nil {
		return fmt.Errorf("failed to check for bastion instance: %w", err)
	}

	if !ready {
		return &ctrlerror.RequeueAfterError{
			RequeueAfter: 10 * time.Second,
			Cause:        errors.New("bastion instance not ready yet"),
		}
	}

	err = ensureSecurityGroupRules(aliCloudECSClient, opt, shootSecurityGroupId, bastion, securityGroupID)
	if err != nil {
		return err
	}

	publicIP, err := aliCloudECSClient.AllocatePublicIp(instanceID)
	if err != nil {
		return err
	}

	endpoints, err := getInstanceEndpoints(aliCloudECSClient, opt, publicIP.IpAddress)
	if err != nil {
		return err
	}

	if !endpoints.Ready() {
		return &ctrlerror.RequeueAfterError{
			// requeue rather soon, so that the user (most likely gardenctl eventually)
			// doesn't have to wait too long for the public endpoint to become available
			RequeueAfter: 5 * time.Second,
			Cause:        fmt.Errorf("bastion instance has no public/private endpoints yet"),
		}
	}

	// once a public endpoint is available, publish the endpoint on the
	// Bastion resource to notify upstream about the ready instance
	patch := client.MergeFrom(bastion.DeepCopy())
	bastion.Status.Ingress = endpoints.public
	return a.client.Status().Patch(ctx, bastion, patch)
}

// IngressReady returns true if either an IP or a hostname or both are set.
func IngressReady(ingress *corev1.LoadBalancerIngress) bool {
	return ingress != nil && (ingress.Hostname != "" || ingress.IP != "")
}

// addressToIngress converts the IP address into a
// corev1.LoadBalancerIngress resource. If both arguments are nil, then
// nil is returned.
func addressToIngress(dnsName *string, ipAddress *string) *corev1.LoadBalancerIngress {
	var ingress *corev1.LoadBalancerIngress

	if ipAddress != nil || dnsName != nil {
		ingress = &corev1.LoadBalancerIngress{}
		if dnsName != nil {
			ingress.Hostname = *dnsName
		}

		if ipAddress != nil {
			ingress.IP = *ipAddress
		}
	}

	return ingress
}

func getInstanceEndpoints(c aliclient.ECS, opt *Options, ip string) (*bastionEndpoints, error) {
	response, err := c.GetInstances(opt.BastionInstanceName)
	if err != nil {
		return nil, err
	}

	if response == nil {
		return nil, fmt.Errorf("compute instance can't be nil")
	}

	if len(response.Instances.Instance) == 0 || response.Instances.Instance[0].Status != "Running" {
		return nil, fmt.Errorf("compute instance not ready yet")
	}

	endpoints := &bastionEndpoints{}
	instance := response.Instances.Instance[0]
	internalIP := instance.VpcAttributes.PrivateIpAddress.IpAddress[0]

	if ingress := addressToIngress(nil, &internalIP); ingress != nil {
		endpoints.private = ingress
	}

	if ingress := addressToIngress(nil, &ip); ingress != nil {
		endpoints.public = ingress
	}

	return endpoints, nil
}

func ensureComputeInstance(c aliclient.ECS, log logr.Logger, opt *Options, securityGroupID, imageID, vSwitchId, zoneID, instanceTypeID string) (string, error) {
	response, err := c.GetInstances(opt.BastionInstanceName)
	if err != nil {
		return "", err
	}

	if len(response.Instances.Instance) > 0 && response.Instances.Instance[0].InstanceName == opt.BastionInstanceName {
		return response.Instances.Instance[0].InstanceId, nil
	}

	log.Info("creating new bastion compute instance")

	instance, err := c.CreateInstances(opt.BastionInstanceName, securityGroupID, imageID, vSwitchId, zoneID, instanceTypeID, opt.UserData)
	if err != nil {
		return "", err
	}

	return instance.InstanceIdSets.InstanceIdSet[0], nil
}

func ensureSecurityGroup(c aliclient.ECS, securityGroupName, vpcID string, log logr.Logger) (string, error) {
	response, err := c.GetSecurityGroup(securityGroupName)
	if err != nil {
		return "", err
	}

	if len(response.SecurityGroups.SecurityGroup) > 0 && response.SecurityGroups.SecurityGroup[0].SecurityGroupName == securityGroupName {
		log.Info("Security Group found", "security group", securityGroupName)
		return response.SecurityGroups.SecurityGroup[0].SecurityGroupId, nil
	}

	log.Info("creating Security Group")

	createResponse, err := c.CreateSecurityGroups(vpcID, securityGroupName)
	if err != nil {
		return "", err
	}

	return createResponse.SecurityGroupId, nil
}

func ensureSecurityGroupRules(c aliclient.ECS, opt *Options, shootSecurityGroupId string, bastion *extensionsv1alpha1.Bastion, securityGroupId string) error {
	// ingress permission
	ingressPermissions, err := ingressPermissions(bastion)
	if err != nil {
		return err
	}

	var wantedIngressRules []*ecs.AuthorizeSecurityGroupRequest

	for _, ingressPermission := range ingressPermissions {
		wantedIngressRules = append(wantedIngressRules, ingressAllowSSH(securityGroupId, ingressPermission))
	}

	currentIngressRules, err := c.DescribeSecurityGroupAttribute(describeSecurityGroupAttributeRequest(securityGroupId, "ingress"))
	if err != nil {
		return err
	}

	rulesToAddIngress, rulesToDelete := ingressRulesSymmetricDifference(wantedIngressRules, currentIngressRules.Permissions.Permission)

	for _, rule := range rulesToAddIngress {
		if err := c.CreateIngressRule(&rule); err != nil {
			return fmt.Errorf("failed to add security group rule %s: %w", rule.Description, err)
		}
	}

	for _, rule := range rulesToDelete {
		if err := c.RevokeIngressRule(revokeSecurityGroupRequest(securityGroupId, rule.IpProtocol, rule.PortRange, rule.SourceCidrIp, rule.Ipv6SourceCidrIp)); err != nil {
			return fmt.Errorf("failed to delete security group rule %s: %w", rule.Description, err)
		}
	}

	// egress rules create
	instanceResponse, err := c.GetInstances(opt.BastionInstanceName)
	if err != nil {
		return err
	}

	if len(instanceResponse.Instances.Instance) == 0 || len(instanceResponse.Instances.Instance[0].VpcAttributes.PrivateIpAddress.IpAddress) == 0 {
		return errors.New("bastion instance does not have a private ip")
	}

	privateIP := instanceResponse.Instances.Instance[0].VpcAttributes.PrivateIpAddress.IpAddress[0]

	wantedEgressRules := []*ecs.AuthorizeSecurityGroupEgressRequest{
		egressAllowSSHToWorker(privateIP, securityGroupId, shootSecurityGroupId),
		egressDenyAll(securityGroupId)}

	currentEgressRules, err := c.DescribeSecurityGroupAttribute(describeSecurityGroupAttributeRequest(securityGroupId, "egress"))
	if err != nil {
		return err
	}

	rulesToAddEgress, rulesToDelete := egressRulesSymmetricDifference(wantedEgressRules, currentEgressRules.Permissions.Permission)
	for _, rule := range rulesToAddEgress {
		if err = c.CreateEgressRule(&rule); err != nil {
			return fmt.Errorf("failed to add security group egress rule %s: %w", rule.Description, err)
		}
	}

	for _, rule := range rulesToDelete {
		if err = c.RevokeEgressRule(revokeSecurityGroupEgressRequest(securityGroupId, rule.IpProtocol, rule.PortRange)); err != nil {
			return fmt.Errorf("failed to delete security egress group rule %s: %w", rule.Description, err)
		}
	}
	return nil
}

func isInstanceReady(c aliclient.ECS, opt *Options) (bool, error) {
	response, err := c.GetInstances(opt.BastionInstanceName)
	if err != nil {
		return false, err
	}

	if response == nil || len(response.Instances.Instance) == 0 {
		return false, errors.New("bastion instance not yet created")
	}

	if response.Instances.Instance[0].Status == "Running" {
		return true, nil
	}

	time.Sleep(10 * time.Second)
	return false, errors.New("bastion instance not yet in Running status")
}

func ingressRulesSymmetricDifference(wantedIngressRules []*ecs.AuthorizeSecurityGroupRequest, currentRules []ecs.Permission) ([]ecs.AuthorizeSecurityGroupRequest, []ecs.Permission) {
	var rulesToDelete []ecs.Permission
	for _, currentRule := range currentRules {
		found := false
		for _, wantedRule := range wantedIngressRules {
			if ingressRuleEqual(*wantedRule, currentRule) {
				found = true
				break
			}
		}

		if !found {
			rulesToDelete = append(rulesToDelete, currentRule)
		}

	}

	var rulesToAdd []ecs.AuthorizeSecurityGroupRequest
	for _, wantedRule := range wantedIngressRules {
		found := false
		for _, currentRule := range currentRules {
			if ingressRuleEqual(*wantedRule, currentRule) {
				found = true
				break
			}
		}

		if !found {
			rulesToAdd = append(rulesToAdd, *wantedRule)
		}
	}
	return rulesToAdd, rulesToDelete
}

func ingressRuleEqual(a ecs.AuthorizeSecurityGroupRequest, b ecs.Permission) bool {
	if !equality.Semantic.DeepEqual(a.Description, b.Description) {
		return false
	}

	if !equality.Semantic.DeepEqual(a.IpProtocol, b.IpProtocol) {
		return false
	}

	if !equality.Semantic.DeepEqual(a.PortRange, b.PortRange) {
		return false
	}

	if !equality.Semantic.DeepEqual(a.SourceCidrIp, b.SourceCidrIp) {
		return false
	}

	if !equality.Semantic.DeepEqual(a.Ipv6SourceCidrIp, b.Ipv6SourceCidrIp) {
		return false
	}

	return true
}

func egressRulesSymmetricDifference(wantedIngressRules []*ecs.AuthorizeSecurityGroupEgressRequest, currentRules []ecs.Permission) ([]ecs.AuthorizeSecurityGroupEgressRequest, []ecs.Permission) {
	var rulesToDelete []ecs.Permission
	for _, currentRule := range currentRules {
		found := false
		for _, wantedRule := range wantedIngressRules {
			if egressRuleEqual(*wantedRule, currentRule) {
				found = true
				break
			}
		}

		if !found {
			rulesToDelete = append(rulesToDelete, currentRule)
		}

	}

	var rulesToAdd []ecs.AuthorizeSecurityGroupEgressRequest
	for _, wantedRule := range wantedIngressRules {
		found := false
		for _, currentRule := range currentRules {
			if egressRuleEqual(*wantedRule, currentRule) {
				found = true
				break
			}
		}

		if !found {
			rulesToAdd = append(rulesToAdd, *wantedRule)
		}
	}
	return rulesToAdd, rulesToDelete
}

func egressRuleEqual(a ecs.AuthorizeSecurityGroupEgressRequest, b ecs.Permission) bool {
	if !equality.Semantic.DeepEqual(a.Description, b.Description) {
		return false
	}

	if !equality.Semantic.DeepEqual(a.IpProtocol, b.IpProtocol) {
		return false
	}

	if !equality.Semantic.DeepEqual(a.PortRange, b.PortRange) {
		return false
	}

	if !equality.Semantic.DeepEqual(a.SourceCidrIp, b.SourceCidrIp) {
		return false
	}

	return true
}

func getInfrastructureStatus(ctx context.Context, a *actuator, cluster *extensions.Cluster) (*alicloudapi.InfrastructureStatus, error) {
	var infrastructureStatus *alicloudapi.InfrastructureStatus
	worker := &extensionsv1alpha1.Worker{}
	err := a.client.Get(ctx, client.ObjectKey{Namespace: cluster.ObjectMeta.Name, Name: cluster.Shoot.Name}, worker)
	if err != nil {
		return nil, err
	}

	if worker.Spec.InfrastructureProviderStatus == nil {
		return nil, errors.New("infrastructure provider status must be not empty for worker")
	}

	if infrastructureStatus, err = helper.InfrastructureStatusFromRaw(worker.Spec.InfrastructureProviderStatus); err != nil {
		return nil, err
	}

	if infrastructureStatus.VPC.ID == "" {
		return nil, errors.New("vpc id must be not empty for infrastructure provider status")
	}

	if len(infrastructureStatus.VPC.VSwitches) == 0 || infrastructureStatus.VPC.VSwitches[0].ID == "" || infrastructureStatus.VPC.VSwitches[0].Zone == "" {
		return nil, errors.New("vswitches id must be not empty for infrastructure provider status")
	}

	if len(infrastructureStatus.MachineImages) == 0 || infrastructureStatus.MachineImages[0].ID == "" {
		return nil, errors.New("machineImages id must be not empty for infrastructure provider status")
	}

	// The assumption is that the shoot only has one security group
	if len(infrastructureStatus.VPC.SecurityGroups) == 0 || infrastructureStatus.VPC.SecurityGroups[0].ID == "" {
		return nil, errors.New("shoot securityGroups id must be not empty for infrastructure provider status")
	}

	return infrastructureStatus, nil
}
