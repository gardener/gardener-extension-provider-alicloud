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
	Name      string
	VpcId     string
	CidrBlock string
	Status    *string
}

// VSwitch is the struct for a vswitch object
type VSwitch struct {
	Tags
	Name      string
	VSwitchId string
	VpcId     *string
	CidrBlock string
	ZoneId    string
	Status    *string
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
