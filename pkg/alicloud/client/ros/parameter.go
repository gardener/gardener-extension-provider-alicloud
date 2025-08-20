// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ros

// Parameter is a nested struct in ros response
type Parameter struct {
	ParameterKey   string `json:"ParameterKey" xml:"ParameterKey"`
	ParameterValue string `json:"ParameterValue" xml:"ParameterValue"`
}
