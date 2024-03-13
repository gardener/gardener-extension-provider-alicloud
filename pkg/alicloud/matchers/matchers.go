// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"fmt"
	"reflect"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

// BeSemanticallyEqualTo returns a matcher that checks if a ecs.Permission is semantically equal to the given value.
func BeSemanticallyEqualTo(expected interface{}) types.GomegaMatcher {
	if expected == nil {
		return BeNil()
	}

	switch expected.(type) {
	case *ecs.Permission:
		return beSemanticallyEqualToIpPermission(expected)
	case []*ecs.Permission:
		return genericConsistOfSemanticallyEqual(expected)
	default:
		panic(fmt.Errorf("unknown type for alicloud matcher BeSemanticallyEqualTo(): %T", expected))
	}
}

func genericBeNilOrEqualTo(expected interface{}) types.GomegaMatcher {
	if expected == nil {
		return BeNil()
	}

	return Equal(expected)
}

func genericConsistOfSemanticallyEqual(expected interface{}) types.GomegaMatcher {
	value := reflect.ValueOf(expected)
	if value.Kind() != reflect.Slice {
		panic(fmt.Errorf("invalid type of expected passed to genericConsistOfSemanticallyEqual, only accepting slices: %s", value.Type().String()))
	}

	if value.Len() == 0 {
		return BeEmpty()
	}

	var expectedElements []interface{}

	for i := 0; i < value.Len(); i++ {
		expectedElement := value.Index(i)
		if expectedElement.IsNil() {
			expectedElements = append(expectedElements, BeNil())
		} else {
			expectedElements = append(expectedElements, BeSemanticallyEqualTo(expectedElement.Interface()))
		}
	}

	return ConsistOf(expectedElements)
}
