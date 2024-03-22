// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ros

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// DeleteStack invokes the ros.DeleteStack API synchronously
// api document: https://www.alibabacloud.com/help/doc-detail/132113.htm
func (client *Client) DeleteStack(request *DeleteStackRequest) (response *DeleteStackResponse, err error) {
	response = CreateDeleteStackResponse()
	err = client.DoAction(request, response)
	return
}

// DeleteStackRequest is the request struct for api DeleteStack
type DeleteStackRequest struct {
	*requests.RpcRequest
	RetainAllResources requests.Boolean `position:"Query" name:"RetainAllResources"`
	StackId            string           `position:"Query" name:"StackId"`
}

// DeleteStackResponse is the response struct for api DeleteStack
type DeleteStackResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateDeleteStackRequest creates a request to invoke DeleteStack API
func CreateDeleteStackRequest() (request *DeleteStackRequest) {
	request = &DeleteStackRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("ROS", "2019-09-10", "DeleteStack", "ROS", "openAPI")
	return
}

// CreateDeleteStackResponse creates a response to parse from DeleteStack response
func CreateDeleteStackResponse() (response *DeleteStackResponse) {
	response = &DeleteStackResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
