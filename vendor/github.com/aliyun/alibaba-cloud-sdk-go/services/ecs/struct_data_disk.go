package ecs

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

// DataDisk is a nested struct in ecs response
type DataDisk struct {
	PerformanceLevel     string `json:"PerformanceLevel" xml:"PerformanceLevel"`
	Description          string `json:"Description" xml:"Description"`
	SnapshotId           string `json:"SnapshotId" xml:"SnapshotId"`
	Device               string `json:"Device" xml:"Device"`
	Size                 int    `json:"Size" xml:"Size"`
	DiskName             string `json:"DiskName" xml:"DiskName"`
	Category             string `json:"Category" xml:"Category"`
	DeleteWithInstance   bool   `json:"DeleteWithInstance" xml:"DeleteWithInstance"`
	Encrypted            string `json:"Encrypted" xml:"Encrypted"`
	ProvisionedIops      int64  `json:"ProvisionedIops" xml:"ProvisionedIops"`
	BurstingEnabled      bool   `json:"BurstingEnabled" xml:"BurstingEnabled"`
	AutoSnapshotPolicyId string `json:"AutoSnapshotPolicyId" xml:"AutoSnapshotPolicyId"`
}