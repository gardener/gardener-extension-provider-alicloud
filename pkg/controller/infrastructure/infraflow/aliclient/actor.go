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

import (
	"context"
	"fmt"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/log"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
)

type Actor interface {
	CreateVpc(ctx context.Context, vpc *VPC) (*VPC, error)
}

type actor struct {
	vpcClient    alicloudclient.VPC
	Logger       logr.Logger
	PollInterval time.Duration
}

var _ Actor = &actor{}

func NewActor(accessKeyID, secretAccessKey, region string) (Actor, error) {

	clientFactory := alicloudclient.NewClientFactory()
	vpcClient, err := clientFactory.NewVPCClient(region, accessKeyID, secretAccessKey)
	if err != nil {
		return nil, err
	}

	return &actor{
		vpcClient:    vpcClient,
		Logger:       log.Log.WithName("alicloud-client"),
		PollInterval: 5 * time.Second,
	}, nil
}

func (c *actor) CreateVpc(ctx context.Context, desired *VPC) (*VPC, error) {
	createVpcReq := vpc.CreateCreateVpcRequest()
	createVpcReq.VpcName = desired.Name
	createVpcReq.CidrBlock = desired.CidrBlock
	createVpcReq.RegionId = desired.Region

	createVPCsResp, err := c.vpcClient.CreateVpc(createVpcReq)
	if err != nil {
		return nil, fmt.Errorf("fail to create vpc, %w", err)
	}

	describeVpcsReq := vpc.CreateDescribeVpcsRequest()
	describeVpcsReq.VpcId = createVPCsResp.VpcId

	err = wait.PollUntil(5*time.Second, func() (bool, error) {
		describeVpcsResp, err := c.vpcClient.DescribeVpcs(describeVpcsReq)
		if err != nil {
			return false, err
		}

		if describeVpcsResp.Vpcs.Vpc[0].Status != "Available" {
			return false, nil
		}
		desired.VpcId = describeVpcsResp.Vpcs.Vpc[0].VpcId

		return true, nil
	}, ctx.Done())

	if err != nil {
		return nil, fmt.Errorf("vpc not Available , %w", err)
	}

	return desired, nil
}
