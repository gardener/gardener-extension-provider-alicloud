package alidns

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

// ConfigInDescribeDnsGtmInstance is a nested struct in alidns response
type ConfigInDescribeDnsGtmInstance struct {
	Ttl                  int                                 `json:"Ttl" xml:"Ttl"`
	AlertGroup           string                              `json:"AlertGroup" xml:"AlertGroup"`
	CnameType            string                              `json:"CnameType" xml:"CnameType"`
	StrategyMode         string                              `json:"StrategyMode" xml:"StrategyMode"`
	InstanceName         string                              `json:"InstanceName" xml:"InstanceName"`
	PublicCnameMode      string                              `json:"PublicCnameMode" xml:"PublicCnameMode"`
	PublicUserDomainName string                              `json:"PublicUserDomainName" xml:"PublicUserDomainName"`
	PubicZoneName        string                              `json:"PubicZoneName" xml:"PubicZoneName"`
	PublicRr             string                              `json:"PublicRr" xml:"PublicRr"`
	AlertConfig          AlertConfigInDescribeDnsGtmInstance `json:"AlertConfig" xml:"AlertConfig"`
}