// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
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
	EnableIpv6    bool
	Ipv6CidrBlock string
	Status        *string
}

// VSwitch is the struct for a vswitch object
type VSwitch struct {
	Tags
	Name            string
	VSwitchId       string
	VpcId           *string
	CidrBlock       string
	ZoneId          string
	EnableIpv6      bool
	Ipv6CidrkSubnet *int
	Ipv6CidrBlock   string
	Status          *string
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

// IPV6Gateway is the struct for a ipv6 gateway object
type IPV6Gateway struct {
	Tags
	Name          string
	VpcId         string
	IPV6GatewayId string
	Status        *string
}
