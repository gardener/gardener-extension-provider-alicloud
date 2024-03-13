// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"flag"
)

const (
	allCIDR     = "0.0.0.0/0"
	vpcCIDR     = "10.250.0.0/16"
	workersCIDR = "10.250.0.0/21"
	podCIDR     = "100.96.0.0/11"

	natGatewayCIDR = "10.250.128.0/21" // Enhanced NatGateway need bind with VSwitch, natGatewayCIDR is used for this VSwitch
	natGatewayType = "Enhanced"

	secretName      = "cloudprovider"
	availableStatus = "Available"
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
		"ap-southeast-1": "m-t4nf5uqofn0vvqjracjy",
	}

	return regionImageMap[region]
}

func getSingleZone(region string) string {
	regionZoneMap := map[string]string{
		"cn-shanghai":    "cn-shanghai-g",
		"ap-southeast-2": "ap-southeast-2a",
		"eu-central-1":   "eu-central-1a",
		"ap-southeast-1": "ap-southeast-1a",
	}

	return regionZoneMap[region]
}
