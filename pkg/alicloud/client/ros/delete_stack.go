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
