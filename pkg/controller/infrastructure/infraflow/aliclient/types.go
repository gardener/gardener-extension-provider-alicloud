// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aliclient

// Factory creates instances of Interface.
type Factory interface {
	// NewClient creates a new instance of Actor for the given alicloud credentials and region.
	NewActor(accessKeyID, secretAccessKey, region string) (Actor, error)
}

// FactoryFunc is a function that implements Factory.
type FactoryFunc func(accessKeyID, secretAccessKey, region string) (Actor, error)

// NewActor creates a new instance of Actor for the given Alicloud credentials and region.
func (f FactoryFunc) NewActor(accessKeyID, secretAccessKey, region string) (Actor, error) {
	return f(accessKeyID, secretAccessKey, region)
}

// VPC is the struct for a vpc object
type VPC struct {
	Tags
	Name          string
	VpcId         string
	CidrBlock     string
	Status        *string
	Ipv6CidrBlock string // IPv6 CIDR for VPC (like "2408:xxxx::/56"), empty for not enabled
}

// VSwitch is the struct for a vswitch object
type VSwitch struct {
	Tags
	Name          string
	VSwitchId     string
	VpcId         *string
	CidrBlock     string
	ZoneId        string
	Status        *string
	Ipv6CidrBlock string // IPv6 CIDR for vswitch (like "2408:xxxx:0:N::/64"), empty for not enabled
}

// NatGateway is the struct for a nat gateway object
type NatGateway struct {
	Tags
	Name               string
	NatGatewayId       string
	VpcId              *string
	VswitchId          *string
	Status             *string
	AvailableVSwitches []string
	SNATTableIDs       []string
}

// EIP is the struct for a eip object
type EIP struct {
	Tags
	Name               string
	Bandwidth          string
	InternetChargeType string
	ZoneId             string
	Status             *string
	EipId              string
	InstanceType       *string
	InstanceId         *string
	IpAddress          string
}

// SNATEntry is the struct for a snat entry object
type SNATEntry struct {
	Name         string
	NatGatewayId string
	VSwitchId    string
	IpAddress    string
	SnatTableId  string
	SnatEntryId  string
	Status       *string
}

// SecurityGroup is the struct for a SecurityGroup object
type SecurityGroup struct {
	Tags
	Name            string
	VpcId           string
	Description     string
	SecurityGroupId string
	Status          *string
	Rules           []*SecurityGroupRule
}

// SecurityGroupRule is the struct for a SecurityGroupRule object
type SecurityGroupRule struct {
	SecurityGroupRuleId string
	Policy              string
	Priority            string
	IpProtocol          string
	PortRange           string
	DestCidrIp          string
	SourceCidrIp        string
	Direction           string
}

// RouteTable is the struct for a route table object.
// RouteTableType is "System" for the VPC default route table, "Custom" for user-created ones.
// The System route table's VSwitchIds is always empty — unassociated VSwitches use it implicitly.
type RouteTable struct {
	Tags
	Name           string
	RouteTableId   string
	VpcId          string
	RouteTableType string // "System" or "Custom"
	VSwitchIds     []string
	Status         *string
}

// RouteEntry is the struct for a route entry in a route table
type RouteEntry struct {
	RouteEntryId         string
	RouteTableId         string
	Name                 string
	DestinationCidrBlock string
	NextHopType          string
	NextHopId            string
	Status               *string
}

// IPv6Gateway is the struct for an IPv6 gateway object
type IPv6Gateway struct {
	Tags
	Name          string
	Ipv6GatewayId string
	VpcId         string
	Status        *string
}

// NLBInfo is the struct for an NLB (Network Load Balancer) instance
type NLBInfo struct {
	LoadBalancerId string
	Name           string
	VpcId          string
	Status         *string
}
