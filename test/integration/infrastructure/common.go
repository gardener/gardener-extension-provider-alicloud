// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package infrastructure

import "flag"

const (
	allCIDR     = "0.0.0.0/0"
	vpcCIDR     = "10.250.0.0/16"
	workersCIDR = "10.250.0.0/21"
	podCIDR     = "100.96.0.0/11"

	natGatewayCIDR = "10.250.128.0/21" // Enhanced NatGateway need bind with VSwitch, natGatewayCIDR is used for this VSwitch
	natGatewayType = "Enhanced"

	secretName      = "cloudprovider"
	availableStatus = "Available"

	eipBandwith = 200
)

var (
	accessKeyID          = flag.String("access-key-id", "", "Alicloud access key id")
	accessKeySecret      = flag.String("access-key-secret", "", "Alicloud access key secret")
	region               = flag.String("region", "", "Alicloud region")
	enableEncryptedImage = flag.Bool("enable-encrypted-image", false, "Enable encrypted image or not")
)

func validateFlags() {
	if len(*accessKeyID) == 0 {
		panic("need an Alicloud access key id")
	}
	if len(*accessKeySecret) == 0 {
		panic("need an Alicloud access key secret")
	}
	if len(*region) == 0 {
		panic("need an Alicloud region")
	}
}

func getImageId(region string) string {
	regionImageMap := map[string]string{
		"cn-shanghai":    "m-uf6a3012pcuemma21nfk",
		"ap-southeast-2": "m-p0w8c5rj528oj84nlise",
		"eu-central-1":   "m-gw83xpc3q3yzpoahhckf",
	}

	return regionImageMap[region]
}

func getSingleZone(region string) string {
	regionZoneMap := map[string]string{
		"cn-shanghai":    "cn-shanghai-g",
		"ap-southeast-2": "ap-southeast-2a",
		"eu-central-1":   "eu-central-1a",
	}

	return regionZoneMap[region]
}
