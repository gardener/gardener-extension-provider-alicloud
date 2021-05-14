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

package ros

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// ListStacksRequest is the request struct for api ListStacks
type ListStacksRequest struct {
	*requests.RpcRequest
	StackName *[]string `position:"Query" name:"StackName"  type:"Repeated"`
}

// ListStacksResponse is the response struct for api ListStacks
type ListStacksResponse struct {
	*responses.BaseResponse
	PageNumber int     `json:"PageNumber" xml:"PageNumber"`
	PageSize   int     `json:"PageSize" xml:"PageSize"`
	RequestId  string  `json:"RequestId" xml:"RequestId"`
	TotalCount int     `json:"TotalCount" xml:"TotalCount"`
	Stacks     []Stack `json:"Stacks" xml:"Stacks"`
}

// ListStacks invokes the ros.ListStacks API synchronously
// api document: https://help.aliyun.com/api/ros/liststacks.html
func (client *Client) ListStacks(request *ListStacksRequest) (response *ListStacksResponse, err error) {
	response = CreateListStacksResponse()
	err = client.DoAction(request, response)
	return
}

// CreateListStacksRequest creates a request to invoke ListStacks API
func CreateListStacksRequest() (request *ListStacksRequest) {
	request = &ListStacksRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("ROS", "2019-09-10", "ListStacks", "ROS", "openAPI")
	return
}

// CreateListStacksResponse creates a response to parse from ListStacks response
func CreateListStacksResponse() (response *ListStacksResponse) {
	response = &ListStacksResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
