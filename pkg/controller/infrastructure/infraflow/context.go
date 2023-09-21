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

package infraflow

import (
	"fmt"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	aliapi "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
)

const (
	// TagKeyName is the name tag key
	TagKeyName = "Name"
	// TagKeyClusterTemplate is the template for the cluster tag key
	TagKeyClusterTemplate = "kubernetes.io/cluster/%s"
	// TagKeyRolePublicELB is the tag key for the public ELB
	TagKeyRolePublicELB = "kubernetes.io/role/elb"
	// TagKeyRolePrivateELB is the tag key for the internal ELB
	TagKeyRolePrivateELB = "kubernetes.io/role/internal-elb"
	// TagValueCluster is the tag value for the cluster tag
	TagValueCluster = "1"
	// TagValueELB is the tag value for the ELB tag keys
	TagValueELB = "1"

	// ChildIdZones is the child key for the zones
	ChildIdZones = "Zones"

	// IdentifierVPC is the key for the VPC id
	IdentifierVPC = "VPC"
	// IdentifierZoneVSwitch is the key for the id of vswitch
	IdentifierNatGatewayVSwitch = "NatGatewayVSwitch"
	IdentifierZoneVSwitch       = "VSwitch"
	IdentifierNatGateway        = "NatGateway"
	// IdentifierZoneNATGWElasticIP is the key for the id of the elastic IP resource used for the NAT gateway
	IdentifierZoneNATGWElasticIP = "NATGatewayElasticIP"

	// IdentifierDHCPOptions is the key for the id of the DHCPOptions resource
	IdentifierDHCPOptions = "DHCPOptions"
	// IdentifierDefaultSecurityGroup is the key for the id of the default security group
	IdentifierDefaultSecurityGroup = "DefaultSecurityGroup"
	// IdentifierInternetGateway is the key for the id of the internet gateway resource
	IdentifierInternetGateway = "InternetGateway"
	// IdentifierMainRouteTable is the key for the id of the main route table
	IdentifierMainRouteTable = "MainRouteTable"
	// IdentifierNodesSecurityGroup is the key for the id of the nodes security group
	IdentifierNodesSecurityGroup = "NodesSecurityGroup"
	// IdentifierZoneSubnetWorkers is the key for the id of the workers subnet
	IdentifierZoneSubnetWorkers = "SubnetWorkers"
	// IdentifierZoneSubnetPublic is the key for the id of the public utility subnet
	IdentifierZoneSubnetPublic = "SubnetPublicUtility"
	// IdentifierZoneSubnetPrivate is the key for the id of the private utility subnet
	IdentifierZoneSubnetPrivate = "SubnetPrivateUtility"
	// IdentifierZoneSuffix is the key for the suffix used for a zone
	IdentifierZoneSuffix = "Suffix"

	// IdentifierZoneNATGateway is the key for the id of the NAT gateway resource
	IdentifierZoneNATGateway = "NATGateway"
	// IdentifierZoneRouteTable is the key for the id of route table of the zone
	IdentifierZoneRouteTable = "ZoneRouteTable"
	// IdentifierZoneSubnetPublicRouteTableAssoc is the key for the id of the public route table association resource
	IdentifierZoneSubnetPublicRouteTableAssoc = "SubnetPublicRouteTableAssoc"
	// IdentifierZoneSubnetPrivateRouteTableAssoc is the key for the id of the private c route table association resource
	IdentifierZoneSubnetPrivateRouteTableAssoc = "SubnetPrivateRouteTableAssoc"
	// IdentifierZoneSubnetWorkersRouteTableAssoc is key for the id of the workers route table association resource
	IdentifierZoneSubnetWorkersRouteTableAssoc = "SubnetWorkersRouteTableAssoc"
	// IdentifierVpcIPv6CidrBlock is the IPv6 CIDR block attached to the vpc
	IdentifierVpcIPv6CidrBlock = "VPCIPv6CidrBlock"
	// NameIAMRole is the key for the name of the IAM role
	NameIAMRole = "IAMRoleName"
	// NameIAMInstanceProfile is the key for the name of the IAM instance profile
	NameIAMInstanceProfile = "IAMInstanceProfileName"
	// NameIAMRolePolicy is the key for the name of the IAM role policy
	NameIAMRolePolicy = "IAMRolePolicyName"
	// NameKeyPair is the key for the name of the EC2 key pair resource
	NameKeyPair = "KeyPair"
	// ARNIAMRole is the key for the ARN of the IAM role
	ARNIAMRole = "IAMRoleARN"
	// KeyPairFingerprint is the key to store the fingerprint of the key pair
	KeyPairFingerprint = "KeyPairFingerprint"
	// KeyPairSpecFingerprint is the key to store the fingerprint of the public key from the spec
	KeyPairSpecFingerprint = "KeyPairSpecFingerprint"

	// ChildIdVPCEndpoints is the child key for the VPC endpoints
	ChildIdVPCEndpoints = "VPCEndpoints"

	// ObjectMainRouteTable is the object key used for caching the main route table object
	ObjectMainRouteTable = "MainRouteTable"
	// ObjectZoneRouteTable is the object key used for caching the zone route table object
	ObjectZoneRouteTable = "ZoneRouteTable"

	// MarkerMigratedFromTerraform is the key for marking the state for successful state migration from Terraformer
	MarkerMigratedFromTerraform = "MigratedFromTerraform"
	// MarkerTerraformCleanedUp is the key for marking the state for successful cleanup of Terraformer resources.
	MarkerTerraformCleanedUp = "TerraformCleanedUp"
	// MarkerLoadBalancersAndSecurityGroupsDestroyed is the key for marking the state that orphan load balancers
	// and security groups have already been destroyed
	MarkerLoadBalancersAndSecurityGroupsDestroyed = "LoadBalancersAndSecurityGroupsDestroyed"
)

// FlowContext contains the logic to reconcile or delete the AWS infrastructure.
type FlowContext struct {
	shared.BasicFlowContext
	state       shared.Whiteboard
	namespace   string
	infraSpec   extensionsv1alpha1.InfrastructureSpec
	config      *aliapi.InfrastructureConfig
	credentials *alicloud.Credentials
	commonTags  aliclient.Tags
	updater     aliclient.Updater
	actor       aliclient.Actor
}

// NewFlowContext creates a new FlowContext object
func NewFlowContext(log logr.Logger, credentials *alicloud.Credentials,
	infra *extensionsv1alpha1.Infrastructure, config *aliapi.InfrastructureConfig,
	oldState shared.FlatMap, persistor shared.FlowStatePersistor) (*FlowContext, error) {

	actor, err := aliclient.NewActor(credentials.AccessKeyID, credentials.AccessKeySecret, infra.Spec.Region)
	if err != nil {
		return nil, err
	}
	updater := aliclient.NewUpdater(actor)
	whiteboard := shared.NewWhiteboard()
	if oldState != nil {
		whiteboard.ImportFromFlatMap(oldState)
	}

	flowContext := &FlowContext{
		BasicFlowContext: *shared.NewBasicFlowContext(log, whiteboard, persistor),
		state:            whiteboard,
		namespace:        infra.Namespace,
		infraSpec:        infra.Spec,
		config:           config,
		updater:          updater,
		actor:            actor,
	}
	flowContext.commonTags = aliclient.Tags{
		flowContext.tagKeyCluster(): TagValueCluster,
		TagKeyName:                  infra.Namespace,
	}
	if config.Networks.VPC.ID != nil {
		flowContext.state.SetPtr(IdentifierVPC, config.Networks.VPC.ID)
	}
	return flowContext, nil
}

func (c *FlowContext) tagKeyCluster() string {
	return fmt.Sprintf(TagKeyClusterTemplate, c.namespace)
}

func (c *FlowContext) hasVPC() bool {
	return !c.state.IsAlreadyDeleted(IdentifierVPC)
}

func (c *FlowContext) hasNatGateway() bool {
	return !c.state.IsAlreadyDeleted(IdentifierNatGateway)
}

func (c *FlowContext) commonTagsWithSuffix(suffix string) aliclient.Tags {
	tags := c.commonTags.Clone()
	tags[TagKeyName] = fmt.Sprintf("%s-%s", c.namespace, suffix)
	return tags
}

func (c *FlowContext) getZoneSuffix(zoneName string) string {
	zoneChild := c.state.GetChild(ChildIdZones).GetChild(zoneName)
	if suffix := zoneChild.Get(IdentifierZoneSuffix); suffix != nil {
		return *suffix
	}
	zones := c.state.GetChild(ChildIdZones)
	existing := sets.New[string]()
	for _, key := range zones.GetChildrenKeys() {
		otherChild := zones.GetChild(key)
		if suffix := otherChild.Get(IdentifierZoneSuffix); suffix != nil {
			existing.Insert(*suffix)
		}
	}
	for i := 0; ; i++ {
		suffix := fmt.Sprintf("z%d", i)
		if !existing.Has(suffix) {
			zoneChild.Set(IdentifierZoneSuffix, suffix)
			return suffix
		}
	}
}

func (c *FlowContext) getAllVSwitchid() []string {
	ids := []string{}
	zones := c.state.GetChild(ChildIdZones)
	for _, key := range zones.GetChildrenKeys() {
		theChild := zones.GetChild(key)
		if switchId := theChild.Get(IdentifierZoneVSwitch); switchId != nil {
			ids = append(ids, *switchId)
		}
	}
	return ids

}

func (c *FlowContext) clusterTags() aliclient.Tags {
	tags := aliclient.Tags{}
	tags[c.tagKeyCluster()] = TagValueCluster
	return tags
}
