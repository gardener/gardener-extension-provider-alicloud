// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
// api document: https://www.alibabacloud.com/help/doc-detail/132117.htm
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
