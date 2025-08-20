// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ros

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// CreateStack invokes the ros.CreateStack API synchronously
// api document: https://www.alibabacloud.com/help/doc-detail/132086.htm
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

// Tag is a repeated param struct in CreateStackRequest
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
