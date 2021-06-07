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

	natGatewayCIDR = "10.250.128.0/21" // Enhanced NatGateway need bind with VSwitch, natGatewayCIDR is used for this VSwitch
	natGatewayType = "Enhanced"

	secretName = "cloudprovider"

	availableStatus = "Available"

	sshPublicKey       = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDSb1DJfupnWTfKJ0fmRGgnSx8A2/pRd5oC49qE1jFX+/J9L01jUyLc5sBKZXVkfU5q5h0JfbkhJXSIkzqE+rNPnJBI4e+8Lo2TVWLAvVZRA9Fg9Dk3mgkdVB+9qW2mIqtJF5GOWKuk7HkObwpY1pX8kHC/LJVfpNQpVBqWef0WJj6vbyjhlZ3vgRxK9I6wdJzjUYtNsDhvvBTy/IBg/xp82w9T2r3GVfnaTLMeQCW9mPviDKnQsrWMgVb2A0Z4c62EbzzLzQV4ScVJ6JMgOgkMqEPdbnKF8dEQcSu+/DQZoZt56Aeov7T4oamahj9/rIDX+WR1nOcfntIdhCyoB4lISkNFz/MlPC7O8HwJk4P7rojLGNk6xmn6NxY5CJGC2dVxFsb1bmm+fKHAp62mgwEoFZcDyIkcsmnmnID9u0rJNyMz84YUGZ/jEz8LePujDHcXiqgoLsKJ8gNRneISL9+m9s1VK7WxDDIbq8iWzR7XfAVE/GzKpVYkqrWCvjKEeFIDuDUnf3jghQCQMsXnJM7zGWr1tl+Dvl2Avxmj2xyUJXYHbXbl2aM434DgQySnV8JPzYH7EsTmvuhdb8SJIbb/NonFsSM+72HpSzVc083x4B++VL7oP1X8cly62pFVM1fi8sxBio48Hq5SmAUu9T4wUY4J+AKU6osFA/ATlMCIiQ== your_email@example.com"
	sshPublicKeyDigest = "b9b39384513d9300374c98ccc8818a8b"
)

var (
	accessKeyID     = flag.String("access-key-id", "", "Alicloud access key id")
	accessKeySecret = flag.String("access-key-secret", "", "Alicloud access key secret")
	region          = flag.String("region", "", "Alicloud region")
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

func getSingleZone(region string) string {
	regionZoneMap := map[string]string{
		"cn-shanghai":  "cn-shanghai-g",
		"eu-central-1": "eu-central-1a",
	}

	return regionZoneMap[region]
}
