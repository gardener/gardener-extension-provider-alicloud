// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"fmt"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/onsi/gomega/format"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

// beSemanticallyEqualToIpPermission returns a matcher that tests if actual is semantically
// equal to the given ec2.IpPermission
func beSemanticallyEqualToIpPermission(expected interface{}) types.GomegaMatcher {
	return &sgPermissionMatcher{
		expected: expected,
	}
}

type sgPermissionMatcher struct {
	expected interface{}
}

func (m *sgPermissionMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil && m.expected == nil {
		return false, fmt.Errorf("refusing to compare <nil> to <nil>.\nBe explicit and use BeNil() instead. This is to avoid mistakes where both sides of an assertion are erroneously uninitialized")
	}

	expectedPermission, ok := m.expected.(ecs.Permission)
	if !ok {
		expectedPermissionPointer, ok2 := m.expected.(*ecs.Permission)
		if !ok2 {
			return false, fmt.Errorf("refusing to compare expected which is neither a ec2.IpPermission nor a *ec2.IpPermission")
		}
		expectedPermission = *expectedPermissionPointer
	}

	actualPermission, ok := actual.(ecs.Permission)
	if !ok {
		actualPermissionPointer, ok2 := actual.(*ecs.Permission)
		if !ok2 {
			return false, fmt.Errorf("refusing to compare actual which is neither a ec2.IpPermission nor a *ec2.IpPermission")
		}
		actualPermission = *actualPermissionPointer
	}

	return MatchFields(IgnoreExtras, Fields{
		"IpProtocol":   genericBeNilOrEqualTo(expectedPermission.IpProtocol),
		"Direction":    genericBeNilOrEqualTo(expectedPermission.Direction),
		"Policy":       genericBeNilOrEqualTo(expectedPermission.Policy),
		"PortRange":    genericBeNilOrEqualTo(expectedPermission.PortRange),
		"Priority":     genericBeNilOrEqualTo(expectedPermission.Priority),
		"SourceCidrIp": genericBeNilOrEqualTo(expectedPermission.SourceCidrIp),
	}).Match(actualPermission)
}

func (m *sgPermissionMatcher) FailureMessage(actual interface{}) (message string) {
	return format.MessageWithDiff(fmt.Sprintf("%+v", actual), "to equal", fmt.Sprintf("%+v", m.expected))
}

func (m *sgPermissionMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.MessageWithDiff(fmt.Sprintf("%+v", actual), "not to equal", fmt.Sprintf("%+v", m.expected))
}
