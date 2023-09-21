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

type VPC struct {
	Tags
	Name      string
	VpcId     string
	CidrBlock string
	Status    *string
}

type VSwitch struct {
	Tags
	Name      string
	VSwitchId string
	VpcId     *string
	CidrBlock string
	ZoneId    string
	Status    *string
}

type NatGateway struct {
	Tags
	Name               string
	NatGatewayId       string
	VpcId              *string
	VswitchId          *string
	Status             *string
	AvailableVSwitches []string
}

type EIP struct {
	Tags
	Name               string
	Bandwidth          string
	InternetChargeType string
	ZoneId             string
	Status             *string
	EipId              string
}
