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
	"reflect"
	"strconv"
	"time"

	alierrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
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

	CreateVSwitch(ctx context.Context, vsw *VSwitch) (*VSwitch, error)
	GetVSwitch(ctx context.Context, id string) (*VSwitch, error)
	GetVSwitches(ctx context.Context, ids []string) ([]*VSwitch, error)
	FindVSwitchesByTags(ctx context.Context, tags Tags) ([]*VSwitch, error)
	DeleteVSwitch(ctx context.Context, id string) error

	CreateNatGateway(ctx context.Context, ngw *NatGateway) (*NatGateway, error)
	GetNatGateway(ctx context.Context, id string) (*NatGateway, error)
	FindNatGatewayByTags(ctx context.Context, tags Tags) ([]*NatGateway, error)
	FindNatGatewayByVPC(ctx context.Context, vpcId string) (*NatGateway, error)
	DeleteNatGateway(ctx context.Context, id string) error

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

func (c *actor) CreateNatGateway(ctx context.Context, ngw *NatGateway) (*NatGateway, error) {
	req := vpc.CreateCreateNatGatewayRequest()
	req.Name = ngw.Name
	req.VpcId = *ngw.VpcId
	req.VSwitchId = *ngw.VswitchId
	req.NatType = "Enhanced"
	resp, err := callApi(c.vpcClient.CreateNatGateway, req)
	if err != nil {
		return nil, err
	}

	var created *NatGateway
	err = wait.PollUntil(5*time.Second, func() (bool, error) {

		created, err = c.GetNatGateway(ctx, resp.NatGatewayId)
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
		return nil, err
	}

	return created, nil

}

func (c *actor) GetNatGateway(ctx context.Context, id string) (*NatGateway, error) {
	req := vpc.CreateDescribeNatGatewaysRequest()
	req.NatGatewayId = id
	resp, err := c.describeNatGateway(ctx, req)

	return single(resp, err)
}

func (c *actor) FindNatGatewayByVPC(ctx context.Context, vpcId string) (*NatGateway, error) {
	req := vpc.CreateDescribeNatGatewaysRequest()
	req.VpcId = vpcId

	resp, err := c.describeNatGateway(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(resp) != 1 {
		return nil, fmt.Errorf("count of natgateway is not 1")
	}
	return resp[0], nil

}

func (c *actor) DeleteNatGateway(ctx context.Context, id string) error {
	req := vpc.CreateDeleteNatGatewayRequest()
	req.NatGatewayId = id
	req.Force = requests.NewBoolean(true)
	_, err := callApi(c.vpcClient.DeleteNatGateway, req)
	if err != nil {
		return err
	}

	err = wait.PollUntil(5*time.Second, func() (bool, error) {
		current, err := c.GetNatGateway(ctx, id)
		if err != nil {
			return false, err
		}
		if current == nil {
			return true, nil
		}
		return false, nil
	}, ctx.Done())

	if err != nil {
		return err
	}
	return nil
}

func (c *actor) FindNatGatewayByTags(ctx context.Context, tags Tags) ([]*NatGateway, error) {
	return nil, nil
}

func (c *actor) DeleteVSwitch(ctx context.Context, id string) error {
	req := vpc.CreateDeleteVSwitchRequest()
	req.VSwitchId = id

	_, err := callApi(c.vpcClient.DeleteVSwitch, req)
	if err != nil {
		return err
	}

	err = wait.PollUntil(5*time.Second, func() (bool, error) {
		current, err := c.GetVSwitch(ctx, id)
		if err != nil {
			return false, err
		}
		if current == nil {
			return true, nil
		}
		return false, nil
	}, ctx.Done())

	if err != nil {
		return err
	}

	return nil
}

func (c *actor) CreateVSwitch(ctx context.Context, vsw *VSwitch) (*VSwitch, error) {
	req := vpc.CreateCreateVSwitchRequest()
	req.VSwitchName = vsw.Name
	req.VpcId = *vsw.VpcId
	req.CidrBlock = vsw.CidrBlock
	req.ZoneId = vsw.ZoneId

	resp, err := callApi(c.vpcClient.CreateVSwitch, req)

	if err != nil {
		return nil, err
	}

	var created *VSwitch
	err = wait.PollUntil(5*time.Second, func() (bool, error) {

		created, err = c.GetVSwitch(ctx, resp.VSwitchId)
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
		return nil, err
	}

	return created, nil
}
func (c *actor) GetVSwitch(ctx context.Context, id string) (*VSwitch, error) {

	req := vpc.CreateDescribeVSwitchesRequest()
	req.VSwitchId = id
	resp, err := c.describeVSwitches(ctx, req)
	return single(resp, err)
}

func (c *actor) GetVSwitches(ctx context.Context, ids []string) ([]*VSwitch, error) {
	var vswitchList []*VSwitch
	for _, id := range ids {
		vsw, err := c.GetVSwitch(ctx, id)
		if err != nil {
			return nil, err
		}
		vswitchList = append(vswitchList, vsw)
	}

	return vswitchList, nil
}

func (c *actor) FindVSwitchesByTags(ctx context.Context, tags Tags) ([]*VSwitch, error) {
	req := vpc.CreateListTagResourcesRequest()
	req.ResourceType = "VSWITCH"

	var reqTag []vpc.ListTagResourcesTag
	for k, v := range tags {
		reqTag = append(reqTag, vpc.ListTagResourcesTag{Key: k, Value: v})
	}
	req.Tag = &reqTag

	var vswitchList []*VSwitch
	idList, err := c.listTagResources(ctx, req)
	if err != nil {
		return vswitchList, err
	}
	return c.GetVSwitches(ctx, idList)

}

func (c *actor) DeleteVpc(ctx context.Context, id string) error {
	req := vpc.CreateDeleteVpcRequest()
	req.VpcId = id

	_, err := callApi(c.vpcClient.DeleteVpc, req)
	if err != nil {
		return err
	}

	err = wait.PollUntil(5*time.Second, func() (bool, error) {
		current, err := c.GetVpc(ctx, id)
		if err != nil {
			return false, err
		}
		if current == nil {
			return true, nil
		}
		return false, nil
	}, ctx.Done())

	if err != nil {
		return err
	}

	return nil

}

func (c *actor) CreateVpc(ctx context.Context, desired *VPC) (*VPC, error) {
	req := vpc.CreateCreateVpcRequest()
	req.VpcName = desired.Name
	req.CidrBlock = desired.CidrBlock

	resp, err := callApi(c.vpcClient.CreateVpc, req)
	if err != nil {
		return nil, err
	}

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
		return nil, err
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
		theVpc, _ := c.GetVpc(ctx, id)
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

func (c *actor) listTagResources(ctx context.Context, req *vpc.ListTagResourcesRequest) ([]string, error) {

	var theList []vpc.TagResource
	var idList []string
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
			idList = append(idList, item.ResourceId)
		}
	}

	return idList, nil
}

func (c *actor) describeNatGateway(ctx context.Context, req *vpc.DescribeNatGatewaysRequest) ([]*NatGateway, error) {
	var ngwList []*NatGateway

	respList, err := call_describe(c.vpcClient.DescribeNatGateways, req)
	if err != nil {
		return ngwList, err
	}
	var theList []vpc.NatGateway
	for _, resp := range respList {
		theList = append(theList, resp.NatGateways.NatGateway...)
	}

	for _, item := range theList {
		ngw, err := c.fromNatGateway(item)
		if err == nil && ngw != nil {
			ngwList = append(ngwList, ngw)
		}
	}

	return ngwList, nil

}

func (c *actor) describeVpcs(ctx context.Context, req *vpc.DescribeVpcsRequest) ([]*VPC, error) {
	var vpcList []*VPC

	respList, err := call_describe(c.vpcClient.DescribeVpcs, req)
	if err != nil {
		return vpcList, err
	}
	var theList []vpc.Vpc
	for _, resp := range respList {
		theList = append(theList, resp.Vpcs.Vpc...)
	}

	for _, item := range theList {
		vpc, err := c.fromVpc(item)
		if err == nil && vpc != nil {
			vpcList = append(vpcList, vpc)
		}
	}

	return vpcList, nil

}

func (c *actor) describeVSwitches(ctx context.Context, req *vpc.DescribeVSwitchesRequest) ([]*VSwitch, error) {

	var vswitchList []*VSwitch

	respList, err := call_describe(c.vpcClient.DescribeVSwitches, req)
	if err != nil {
		return vswitchList, err
	}
	var theList []vpc.VSwitch
	for _, resp := range respList {
		theList = append(theList, resp.VSwitches.VSwitch...)
	}

	for _, item := range theList {
		vswitch, err := c.fromVSwitch(item)
		if err == nil && vswitch != nil {
			vswitchList = append(vswitchList, vswitch)
		}
	}

	return vswitchList, nil

}
func (c *actor) fromVSwitch(item vpc.VSwitch) (*VSwitch, error) {
	vswitch := &VSwitch{
		Name:      item.VSwitchName,
		VpcId:     &item.VpcId,
		ZoneId:    item.ZoneId,
		CidrBlock: item.CidrBlock,
		Status:    &item.Status,
		VSwitchId: item.VSwitchId,
	}
	tags := Tags{}
	for _, t := range item.Tags.Tag {
		tags[t.Key] = t.Value
	}
	vswitch.Tags = tags
	return vswitch, nil
}

func (c *actor) fromNatGateway(item vpc.NatGateway) (*NatGateway, error) {
	ngw := &NatGateway{
		Name:         item.Name,
		NatGatewayId: item.NatGatewayId,
		VpcId:        &item.VpcId,
		Status:       &item.Status,
		VswitchId:    &item.NatGatewayPrivateInfo.VswitchId,
	}
	// tags := Tags{}
	// for _, t := range item.Tags.Tag {
	// 	tags[t.Key] = t.Value
	// }
	// ngw.Tags = tags
	return ngw, nil
}

func (c *actor) fromVpc(item vpc.Vpc) (*VPC, error) {
	vpc := &VPC{
		Name:  item.VpcName,
		VpcId: item.VpcId,

		CidrBlock: item.CidrBlock,
		Status:    &item.Status,
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
			if serverErr, ok := err.(*alierrors.ServerError); ok {
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
func call_describe[REQ any, RESP any](call func(req *REQ) (*RESP, error), req *REQ) ([]RESP, error) {
	type1_req_type_name_list := []string{
		"DescribeVpcsRequest",
		"DescribeVSwitchesRequest",
		"DescribeNatGatewaysRequest",
	}

	reqTypeName := reflect.ValueOf(req).Elem().Type().Name()
	if contains(type1_req_type_name_list, reqTypeName) {
		return call_describe_type1(call, req)
	}
	return nil, fmt.Errorf(fmt.Sprintf("can not found suitable describe function for %s", reqTypeName))
}

func call_describe_type1[REQ any, RESP any](call func(req *REQ) (*RESP, error), req *REQ) ([]RESP, error) {

	var theList []RESP
	const PAGE_SIZE = 10
	var cur_page = 1
	reflect.ValueOf(req).Elem().FieldByName("PageSize").SetString(strconv.Itoa(PAGE_SIZE))
	for {
		reflect.ValueOf(req).Elem().FieldByName("PageNumber").SetString(strconv.Itoa(cur_page))
		resp, err := callApi(call, req)
		if err != nil {
			return nil, err
		}
		theList = append(theList, *resp)

		total := int(reflect.ValueOf(*resp).FieldByName("TotalCount").Int())
		total_page := total / PAGE_SIZE
		remainder := total % PAGE_SIZE
		if remainder > 0 {
			total_page += 1
		}
		if cur_page >= total_page {
			break
		}
		cur_page++
	}
	return theList, nil
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

func contains(elems []string, elem string) bool {
	for _, e := range elems {
		if e == elem {
			return true
		}
	}
	return false
}
