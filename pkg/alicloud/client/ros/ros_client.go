// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package ros

import (
	"reflect"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
)

// Client is the sdk client struct, each func corresponds to an OpenAPI
type Client struct {
	sdk.Client
}

// SetClientProperty Set Property by Reflect
func SetClientProperty(client *Client, propertyName string, propertyValue interface{}) {
	v := reflect.ValueOf(client).Elem()
	if v.FieldByName(propertyName).IsValid() && v.FieldByName(propertyName).CanSet() {
		v.FieldByName(propertyName).Set(reflect.ValueOf(propertyValue))
	}
}

// SetEndpointDataToClient Set EndpointMap and ENdpointType
func SetEndpointDataToClient(client *Client) {
	SetClientProperty(client, "EndpointMap", GetEndpointMap())
	SetClientProperty(client, "EndpointType", GetEndpointType())
}

// NewClientWithAccessKey is a shortcut to create sdk client with accesskey
// usage: https://github.com/aliyun/alibaba-cloud-sdk-go/blob/master/docs/2-Client-EN.md
func NewClientWithAccessKey(regionId, accessKeyId, accessKeySecret string) (client *Client, err error) {
	client = &Client{}
	err = client.InitWithAccessKey(regionId, accessKeyId, accessKeySecret)
	SetEndpointDataToClient(client)
	return
}
