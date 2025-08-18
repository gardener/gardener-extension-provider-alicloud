// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ros

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// GetStack invokes the ros.GetStack API synchronously
// api document: https://www.alibabacloud.com/help/doc-detail/132088.htm
func (client *Client) GetStack(request *GetStackRequest) (response *GetStackResponse, err error) {
	response = CreateGetStackResponse()
	err = client.DoAction(request, response)
	return
}

// GetStackRequest is the request struct for api GetStack
type GetStackRequest struct {
	*requests.RpcRequest
	ClientToken string `position:"Query" name:"ClientToken"`
	StackId     string `position:"Query" name:"StackId"`
}

// GetStackResponse is the response struct for api GetStack
type GetStackResponse struct {
	*responses.BaseResponse
	CreateTime          string              `json:"CreateTime" xml:"CreateTime"`
	Description         string              `json:"Description" xml:"Description"`
	DisableRollback     bool                `json:"DisableRollback" xml:"DisableRollback"`
	RegionId            string              `json:"RegionId" xml:"RegionId"`
	RequestId           string              `json:"RequestId" xml:"RequestId"`
	StackId             string              `json:"StackId" xml:"StackId"`
	StackName           string              `json:"StackName" xml:"StackName"`
	Status              string              `json:"Status" xml:"Status"`
	StatusReason        string              `json:"StatusReason" xml:"StatusReason"`
	TemplateDescription string              `json:"TemplateDescription" xml:"TemplateDescription"`
	TimeoutInMinutes    int                 `json:"TimeoutInMinutes" xml:"TimeoutInMinutes"`
	UpdateTime          string              `json:"UpdateTime" xml:"UpdateTime"`
	ParentStackId       string              `json:"ParentStackId" xml:"ParentStackId"`
	Outputs             []map[string]string `json:"Outputs" xml:"Outputs"`
	NotificationURLs    []string            `json:"NotificationURLs" xml:"NotificationURLs"`
	Parameters          []Parameter         `json:"Parameters" xml:"Parameters"`
}

// CreateGetStackRequest creates a request to invoke GetStack API
func CreateGetStackRequest() (request *GetStackRequest) {
	request = &GetStackRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("ROS", "2019-09-10", "GetStack", "ROS", "openAPI")
	return
}

// CreateGetStackResponse creates a response to parse from GetStack response
func CreateGetStackResponse() (response *GetStackResponse) {
	response = &GetStackResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
