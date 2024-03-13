// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
