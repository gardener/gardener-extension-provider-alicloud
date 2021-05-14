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

// CreateStack invokes the ros.CreateStack API synchronously
// api document: https://help.aliyun.com/api/ros/createstack.html
func (client *Client) CreateStack(request *CreateStackRequest) (response *CreateStackResponse, err error) {
	response = CreateCreateStackResponse()
	err = client.DoAction(request, response)
	return
}

// CreateStackRequest is the request struct for api CreateStack
type CreateStackRequest struct {
	*requests.RpcRequest
	ClientToken        string                   `position:"Query" name:"ClientToken"`
	TemplateBody       string                   `position:"Query" name:"TemplateBody"`
	DisableRollback    requests.Boolean         `position:"Query" name:"DisableRollback"`
	TimeoutInMinutes   requests.Integer         `position:"Query" name:"TimeoutInMinutes"`
	OrderSource        string                   `position:"Query" name:"OrderSource"`
	TemplateURL        string                   `position:"Query" name:"TemplateURL"`
	ActivityId         string                   `position:"Query" name:"ActivityId"`
	NotificationURLs   *[]string                `position:"Query" name:"NotificationURLs"  type:"Repeated"`
	StackPolicyURL     string                   `position:"Query" name:"StackPolicyURL"`
	StackName          string                   `position:"Query" name:"StackName"`
	Parameters         *[]CreateStackParameters `position:"Query" name:"Parameters"  type:"Repeated"`
	StackPolicyBody    string                   `position:"Query" name:"StackPolicyBody"`
	DeletionProtection string                   `position:"Query" name:"DeletionProtection"`
	Tags               *[]Tag                   `position:"Query" name:"Tags"  type:"Repeated"`
}

// Tags is a repeated param struct in CreateStackRequest
type Tag struct {
	Value string `name:"Value"`
	Key   string `name:"Key"`
}

// CreateStackParameters is a repeated param struct in CreateStackRequest
type CreateStackParameters struct {
	ParameterValue string `name:"ParameterValue"`
	ParameterKey   string `name:"ParameterKey"`
}

// CreateStackResponse is the response struct for api CreateStack
type CreateStackResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	StackId   string `json:"StackId" xml:"StackId"`
}

// CreateCreateStackRequest creates a request to invoke CreateStack API
func CreateCreateStackRequest() (request *CreateStackRequest) {
	request = &CreateStackRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("ROS", "2019-09-10", "CreateStack", "ROS", "openAPI")
	return
}

// CreateCreateStackResponse creates a response to parse from CreateStack response
func CreateCreateStackResponse() (response *CreateStackResponse) {
	response = &CreateStackResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
