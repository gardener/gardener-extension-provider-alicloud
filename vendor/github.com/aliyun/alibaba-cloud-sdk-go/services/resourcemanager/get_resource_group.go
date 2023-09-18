package resourcemanager

//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
// Code generated by Alibaba Cloud SDK Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// GetResourceGroup invokes the resourcemanager.GetResourceGroup API synchronously
func (client *Client) GetResourceGroup(request *GetResourceGroupRequest) (response *GetResourceGroupResponse, err error) {
	response = CreateGetResourceGroupResponse()
	err = client.DoAction(request, response)
	return
}

// GetResourceGroupWithChan invokes the resourcemanager.GetResourceGroup API asynchronously
func (client *Client) GetResourceGroupWithChan(request *GetResourceGroupRequest) (<-chan *GetResourceGroupResponse, <-chan error) {
	responseChan := make(chan *GetResourceGroupResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.GetResourceGroup(request)
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response
		}
	})
	if err != nil {
		errChan <- err
		close(responseChan)
		close(errChan)
	}
	return responseChan, errChan
}

// GetResourceGroupWithCallback invokes the resourcemanager.GetResourceGroup API asynchronously
func (client *Client) GetResourceGroupWithCallback(request *GetResourceGroupRequest, callback func(response *GetResourceGroupResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *GetResourceGroupResponse
		var err error
		defer close(result)
		response, err = client.GetResourceGroup(request)
		callback(response, err)
		result <- 1
	})
	if err != nil {
		defer close(result)
		callback(nil, err)
		result <- 0
	}
	return result
}

// GetResourceGroupRequest is the request struct for api GetResourceGroup
type GetResourceGroupRequest struct {
	*requests.RpcRequest
	ResourceGroupId string           `position:"Query" name:"ResourceGroupId"`
	IncludeTags     requests.Boolean `position:"Query" name:"IncludeTags"`
}

// GetResourceGroupResponse is the response struct for api GetResourceGroup
type GetResourceGroupResponse struct {
	*responses.BaseResponse
	RequestId     string        `json:"RequestId" xml:"RequestId"`
	ResourceGroup ResourceGroup `json:"ResourceGroup" xml:"ResourceGroup"`
}

// CreateGetResourceGroupRequest creates a request to invoke GetResourceGroup API
func CreateGetResourceGroupRequest() (request *GetResourceGroupRequest) {
	request = &GetResourceGroupRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("ResourceManager", "2020-03-31", "GetResourceGroup", "", "")
	request.Method = requests.POST
	return
}

// CreateGetResourceGroupResponse creates a response to parse from GetResourceGroup response
func CreateGetResourceGroupResponse() (response *GetResourceGroupResponse) {
	response = &GetResourceGroupResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
