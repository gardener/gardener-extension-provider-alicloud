// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infraflow

import (
	"fmt"
	"strings"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
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
	// TagValueCluster is the tag value for the cluster tag
	TagValueCluster = "1"

	// ChildIdZones is the child key for the zones
	ChildIdZones = "Zones"

	// IdentifierVPC is the key for the VPC id
	IdentifierVPC = "VPC"
	// IdentifierNatGatewayVSwitch is the natgateway id for the vswitch
	IdentifierNatGatewayVSwitch = "NatGatewayVSwitch"
	// IdentifierZoneVSwitch is the key for the id of vswitch
	IdentifierZoneVSwitch = "VSwitch"
	// IdentifierNatGateway is the key for the id of natgateway
	IdentifierNatGateway = "NatGateway"
	// IdentifierZoneNATGWElasticIP is the key for the id of the elastic IP resource used for the NAT gateway
	IdentifierZoneNATGWElasticIP = "NATGatewayElasticIP"
	//ZoneNATGWElasticIPAddress is the ipaddress of the elastic IP resource used for the NAT gateway
	ZoneNATGWElasticIPAddress = "NATGatewayElasticIPAddress"
	// IdentifierNodesSecurityGroup is the key for the id of the nodes security group
	IdentifierNodesSecurityGroup = "NodesSecurityGroup"
	// IdentifierIPV6Gateway is the key for the id of ipv6gateway
	IdentifierIPV6Gateway = "IPV6Gateway"
	// IdentifierRouteTable is the key for the id of the custom route table
	IdentifierRouteTable = "RouteTable"

	// IdentifierZoneSuffix is the key for the suffix used for a zone
	IdentifierZoneSuffix = "Suffix"

	// MarkerMigratedFromTerraform is the key for marking the state for successful state migration from Terraformer
	MarkerMigratedFromTerraform = "MigratedFromTerraform"
	// MarkerTerraformCleanedUp is the key for marking the state for successful cleanup of Terraformer resources.
	MarkerTerraformCleanedUp = "TerraformCleanedUp"
)

// FlowContext contains the logic to reconcile or delete the AWS infrastructure.
type FlowContext struct {
	shared.BasicFlowContext
	state      shared.Whiteboard
	namespace  string
	infraSpec  extensionsv1alpha1.InfrastructureSpec
	config     *aliapi.InfrastructureConfig
	commonTags aliclient.Tags
	updater    aliclient.Updater
	actor      aliclient.Actor
	cluster    *extensioncontroller.Cluster
	canDelete  bool
}

// NewFlowContext creates a new FlowContext object
func NewFlowContext(log logr.Logger, credentials *alicloud.Credentials,
	infra *extensionsv1alpha1.Infrastructure, config *aliapi.InfrastructureConfig,
	oldState shared.FlatMap, persistor shared.FlowStatePersistor, cluster *extensioncontroller.Cluster) (*FlowContext, error) {
	actor, err := aliclient.NewActor(credentials.AccessKeyID, credentials.AccessKeySecret, infra.Spec.Region)
	if err != nil {
		return nil, err
	}
	canDelete := false
	updater := aliclient.NewUpdater(actor)
	whiteboard := shared.NewWhiteboard()
	if oldState != nil {
		whiteboard.ImportFromFlatMap(oldState)
	}
	if infra.Annotations != nil && strings.EqualFold(infra.Annotations[aliapi.AnnotationKeyFlowReconcileCanDeleteResource], "true") {
		canDelete = true
	}

	flowContext := &FlowContext{
		BasicFlowContext: *shared.NewBasicFlowContext(log, whiteboard, persistor),
		state:            whiteboard,
		namespace:        infra.Namespace,
		infraSpec:        infra.Spec,
		config:           config,
		updater:          updater,
		actor:            actor,
		cluster:          cluster,
		canDelete:        canDelete,
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

func (c *FlowContext) useCustomRouteTable() bool {
	return c.config.Networks.VPC.UseCustomRouteTable != nil &&
		*c.config.Networks.VPC.UseCustomRouteTable
}

// isOwnedByAnotherShoot returns true if the given tags contain a kubernetes.io/cluster/ tag
// that belongs to a different shoot namespace. Used to exclude resources created by other
// shoots sharing the same user-provided VPC, while still allowing reuse of resources owned
// by the current shoot (e.g. VSwitches).
func (c *FlowContext) isOwnedByAnotherShoot(tags aliclient.Tags) bool {
	ourClusterTag := c.tagKeyCluster()
	for k := range tags {
		if strings.HasPrefix(k, "kubernetes.io/cluster/") && k != ourClusterTag {
			return true
		}
	}
	return false
}

func (c *FlowContext) dualStackEnabled() bool {
	return c.config.DualStack != nil && c.config.DualStack.Enabled
}

// isBYOInfrastructure returns true if all zones use workersVSwitchID (BYO VSwitch mode).
// Validation ensures all zones use the same approach, so checking the first zone suffices.
func (c *FlowContext) isBYOInfrastructure() bool {
	return len(c.config.Networks.Zones) > 0 && c.config.Networks.Zones[0].WorkersVSwitchID != nil
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

func (c *FlowContext) getAllVSwitchids() []string {
	ids := []string{}
	firstZoneName := c.config.Networks.Zones[0].Name
	firstZone := c.getZoneChild(firstZoneName)
	if switchId := firstZone.Get(IdentifierZoneVSwitch); switchId != nil {
		ids = append(ids, *switchId)
	}
	zones := c.state.GetChild(ChildIdZones)
	for _, key := range zones.GetChildrenKeys() {
		if key != firstZoneName {
			theChild := zones.GetChild(key)
			if switchId := theChild.Get(IdentifierZoneVSwitch); switchId != nil {
				ids = append(ids, *switchId)
			}
		}
	}
	return ids
}

func (c *FlowContext) clusterTags() aliclient.Tags {
	tags := aliclient.Tags{}
	tags[c.tagKeyCluster()] = TagValueCluster
	return tags
}

// ExportState is used to export the flatMap data
func (c *FlowContext) ExportState() shared.FlatMap {
	return c.state.ExportAsFlatMap()
}

// getEffectiveIpv6CidrBlock returns the IPv6 /64 subnet index for the given zone.
// If the zone config has an explicit Ipv6CidrBlock, that value is returned.
// Otherwise the zone's position index in config.Networks.Zones is used as a default,
// which is safe for Gardener-managed VPC (user-provided VPC always has non-nil value
// enforced by admission validation).
func (c *FlowContext) getEffectiveIpv6CidrBlock(zoneName string) (int, error) {
	for i, zone := range c.config.Networks.Zones {
		if zone.Name == zoneName {
			if zone.Ipv6CidrBlock != nil {
				return *zone.Ipv6CidrBlock, nil
			}
			return i, nil
		}
	}
	return -1, fmt.Errorf("zone %s not found in config", zoneName)
}
