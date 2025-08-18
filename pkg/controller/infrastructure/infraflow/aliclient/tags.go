// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aliclient

// Tags is map of string key to string values. Duplicate keys are not supported in AWS.
type Tags map[string]string

// Clone creates a copy of the tags aps
func (tags Tags) Clone() Tags {
	cp := Tags{}
	for k, v := range tags {
		cp[k] = v
	}
	return cp
}
