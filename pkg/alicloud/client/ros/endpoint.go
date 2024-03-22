// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ros

var endpointMap = map[string]string{}

var endpointType = "central"

// GetEndpointMap Get Endpoint Data Map
func GetEndpointMap() map[string]string {
	return endpointMap
}

// GetEndpointType Get Endpoint Type Value
func GetEndpointType() string {
	return endpointType
}
