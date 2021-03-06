// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		if ok2 {
			expectedPermission = *expectedPermissionPointer
		} else {
			return false, fmt.Errorf("refusing to compare expected which is neither a ec2.IpPermission nor a *ec2.IpPermission")
		}
	}

	actualPermission, ok := actual.(ecs.Permission)
	if !ok {
		actualPermissionPointer, ok2 := actual.(*ecs.Permission)
		if ok2 {
			actualPermission = *actualPermissionPointer
		} else {
			return false, fmt.Errorf("refusing to compare actual which is neither a ec2.IpPermission nor a *ec2.IpPermission")
		}
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
