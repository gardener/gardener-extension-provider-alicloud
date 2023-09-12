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

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/log"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
)

type Actor interface {
	CreateVpc(ctx context.Context, vpc *VPC) (*VPC, error)
	GetVpc(ctx context.Context, id string) (*VPC, error)
	FindVpcsByTags(ctx context.Context, tags Tags) ([]*VPC, error)
	DeleteVpc(ctx context.Context, id string) error

	CreateVpcTags(ctx context.Context, resources []string, tags Tags, resourceType string) error
	DeleteVpcTags(ctx context.Context, resources []string, tags Tags, resourceType string) error
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

func (c *actor) DeleteVpc(ctx context.Context, id string) error {
	req := vpc.CreateDeleteVpcRequest()
	req.VpcId = id

	_, err := callApi(c.vpcClient.DeleteVpc, req)
	if err != nil {
		return fmt.Errorf("fail to delete vpc, %w", err)
	}

	return nil

}

func (c *actor) CreateVpc(ctx context.Context, desired *VPC) (*VPC, error) {
	req := vpc.CreateCreateVpcRequest()
	req.VpcName = desired.Name
	req.CidrBlock = desired.CidrBlock

	resp, err := callApi(c.vpcClient.CreateVpc, req)
	if err != nil {
		return nil, fmt.Errorf("fail to create vpc, %w", err)
	}

	describeVpcsReq := vpc.CreateDescribeVpcsRequest()
	describeVpcsReq.VpcId = resp.VpcId
	var created *VPC
	err = wait.PollUntil(5*time.Second, func() (bool, error) {

		created, err = c.GetVpc(ctx, resp.VpcId)
		if err != nil {
			return false, err
		}
		if created == nil {
			return false, nil
		}
		if *created.Status != "Available" {
			return false, nil
		}

		return true, nil
	}, ctx.Done())

	if err != nil {
		return nil, fmt.Errorf("vpc not Available , %w", err)
	}

	return created, nil
}

func (c *actor) GetVpc(ctx context.Context, id string) (*VPC, error) {

	req := vpc.CreateDescribeVpcsRequest()
	req.VpcId = id

	resp, err := c.describeVpcs(ctx, req)
	return single(resp, err)
}

func (c *actor) FindVpcsByTags(ctx context.Context, tags Tags) ([]*VPC, error) {
	req := vpc.CreateListTagResourcesRequest()
	req.ResourceType = "VPC"

	var reqTag []vpc.ListTagResourcesTag
	for k, v := range tags {
		reqTag = append(reqTag, vpc.ListTagResourcesTag{Key: k, Value: v})
	}
	req.Tag = &reqTag

	var vpcList []*VPC
	idList, err := c.listTagResources(ctx, req)
	if err != nil {
		return vpcList, err
	}
	for _, id := range idList {
		theVpc, _ := c.GetVpc(ctx, *id)
		if theVpc != nil {
			vpcList = append(vpcList, theVpc)
		}
	}
	return vpcList, nil

}

func (c *actor) CreateVpcTags(ctx context.Context, resources []string, tags Tags, resourceType string) error {
	req := vpc.CreateTagResourcesRequest()
	req.ResourceType = resourceType
	req.ResourceId = &resources

	var reqTag []vpc.TagResourcesTag
	for k, v := range tags {
		reqTag = append(reqTag, vpc.TagResourcesTag{Key: k, Value: v})
	}
	req.Tag = &reqTag

	_, err := callApi(c.vpcClient.TagResources, req)
	return err
}

func (c *actor) DeleteVpcTags(ctx context.Context, resources []string, tags Tags, resourceType string) error {
	req := vpc.CreateUnTagResourcesRequest()
	req.ResourceType = resourceType
	req.ResourceId = &resources

	var reqTag []vpc.UnTagResourcesTag
	for k, v := range tags {
		reqTag = append(reqTag, vpc.UnTagResourcesTag{Key: k, Value: v})
	}
	req.Tag = &reqTag
	_, err := callApi(c.vpcClient.UnTagResources, req)
	return err
}

func (c *actor) listTagResources(ctx context.Context, req *vpc.ListTagResourcesRequest) ([]*string, error) {

	var theList []vpc.TagResource
	var idList []*string
	resp, err := callApi(c.vpcClient.ListTagResources, req)

	if err != nil {
		return idList, err
	}
	theList = append(theList, resp.TagResources.TagResource...)
	for {
		if resp.NextToken == "" {
			break
		} else {
			req.NextToken = resp.NextToken
			resp, err := callApi(c.vpcClient.ListTagResources, req)
			if err == nil {
				theList = append(theList, resp.TagResources.TagResource...)
			}
		}
	}

	for _, item := range theList {
		if !contains(idList, item.ResourceId) {
			idList = append(idList, &item.ResourceId)
		}
	}

	return idList, nil
}

func (c *actor) describeVpcs(ctx context.Context, req *vpc.DescribeVpcsRequest) ([]*VPC, error) {
	var vpcList []*VPC
	resp, err := callApi(c.vpcClient.DescribeVpcs, req)
	if err != nil {
		return vpcList, err
	}

	var theList []vpc.Vpc
	cur_page := 1
	total := resp.TotalCount
	if total > 0 {
		theList = append(theList, resp.Vpcs.Vpc...)
	}
	for {
		if len(theList) >= total {
			break
		}
		cur_page = cur_page + 1
		req.PageNumber = requests.NewInteger(cur_page)
		resp, err := callApi(c.vpcClient.DescribeVpcs, req)
		if err == nil {
			theList = append(theList, resp.Vpcs.Vpc...)
		}
	}

	for _, item := range theList {
		vpc, err := c.fromVpc(ctx, item)
		if err == nil && vpc != nil {
			vpcList = append(vpcList, vpc)
		}
	}
	return vpcList, nil
}

func (c *actor) fromVpc(ctx context.Context, item vpc.Vpc) (*VPC, error) {
	vpc := &VPC{
		Name:  item.VpcName,
		VpcId: item.VpcId,

		CidrBlock:     item.CidrBlock,
		IPv6CidrBlock: item.Ipv6CidrBlock,
		Status:        &item.Status,
	}

	tags := Tags{}
	for _, t := range item.Tags.Tag {
		tags[t.Key] = t.Value
	}
	vpc.Tags = tags

	return vpc, nil
}

func callApi[REQ any, RESP any](call func(req *REQ) (*RESP, error), req *REQ) (*RESP, error) {
	var resp *RESP
	var err error
	try_count := 0
	for {
		need_try := false
		try_count = try_count + 1
		cleanQueryParam(req)
		resp, err = call(req)
		if err != nil {
			if serverErr, ok := err.(*errors.ServerError); ok {
				if serverErr.ErrorCode() == "Throttling.User" && try_count < 5 {
					need_try = true
				}
			}
		}
		if need_try {
			time.Sleep(10 * time.Second)
		} else {
			break
		}
	}
	return resp, err
}

func cleanQueryParam(theReq interface{}) {
	if req, ok := theReq.(*requests.RpcRequest); ok {
		queryParam := req.GetQueryParams()
		delete(queryParam, "Version")
		delete(queryParam, "Action")
		delete(queryParam, "Format")
		delete(queryParam, "Timestamp")
		delete(queryParam, "SignatureMethod")
		delete(queryParam, "SignatureType")
		delete(queryParam, "SignatureVersion")
		delete(queryParam, "SignatureNonce")
		delete(queryParam, "AccessKeyId")
		delete(queryParam, "RegionId")
	}
}

func single[T any](list []*T, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

func contains(elems []*string, elem string) bool {
	for _, e := range elems {
		if e != nil && *e == elem {
			return true
		}
	}
	return false
}
