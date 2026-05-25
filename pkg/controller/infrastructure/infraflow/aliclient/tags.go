// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aliclient

import "strings"

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

// IsGardenerManaged returns true if the tags contain any kubernetes.io/cluster/ key,
// indicating the resource is owned by a Gardener shoot (any shoot).
func IsGardenerManaged(tags Tags) bool {
	for k := range tags {
		if strings.HasPrefix(k, "kubernetes.io/cluster/") {
			return true
		}
	}
	return false
}
