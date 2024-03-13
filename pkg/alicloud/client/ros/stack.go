// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ros

// Stack is a nested struct in ros response
type Stack struct {
	CreateTime       string `json:"CreateTime" xml:"CreateTime"`
	DisableRollback  bool   `json:"DisableRollback" xml:"DisableRollback"`
	RegionId         string `json:"RegionId" xml:"RegionId"`
	StackId          string `json:"StackId" xml:"StackId"`
	StackName        string `json:"StackName" xml:"StackName"`
	Status           string `json:"Status" xml:"Status"`
	StatusReason     string `json:"StatusReason" xml:"StatusReason"`
	TimeoutInMinutes int    `json:"TimeoutInMinutes" xml:"TimeoutInMinutes"`
	ParentStackId    string `json:"ParentStackId" xml:"ParentStackId"`
	UpdateTime       string `json:"UpdateTime" xml:"UpdateTime"`
}
