// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package alicloud

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"k8s.io/utils/pointer"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Credentials are the credentials to access the Alicloud API.
type Credentials struct {
	AccessKeyID     string
	AccessKeySecret string
}

const (
	// AccessKeyID is the data field in a secret where the access key id is stored at.
	AccessKeyID = "accessKeyID"
	// AccessKeySecret is the data field in a secret where the access key secret is stored at.
	AccessKeySecret = "accessKeySecret"

	// DNSAccessKeyID is the data field in a DNS secret where the access key id is stored at.
	DNSAccessKeyID = "ACCESS_KEY_ID"
	// DNSAccessKeySecret is the data field in a DNS secret where the access key secret is stored at.
	DNSAccessKeySecret = "ACCESS_KEY_SECRET"
)

// ReadSecretCredentials reads the Credentials from the given secret.
func ReadSecretCredentials(secret *corev1.Secret, allowDNSKeys bool) (*Credentials, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret %s/%s has no data section", secret.Namespace, secret.Name)
	}

	var altAccessKeyIDKey, altAccessKeySecretKey *string
	if allowDNSKeys {
		altAccessKeyIDKey, altAccessKeySecretKey = pointer.String(DNSAccessKeyID), pointer.StringPtr(DNSAccessKeySecret)
	}

	accessKeyID, ok := getSecretDataValue(secret, AccessKeyID, altAccessKeyIDKey)
	if !ok {
		return nil, fmt.Errorf("secret %s/%s has no access key id", secret.Namespace, secret.Name)
	}

	accessKeySecret, ok := getSecretDataValue(secret, AccessKeySecret, altAccessKeySecretKey)
	if !ok {
		return nil, fmt.Errorf("secret %s/%s has no access key secret", secret.Namespace, secret.Name)
	}

	return &Credentials{
		AccessKeyID:     string(accessKeyID),
		AccessKeySecret: string(accessKeySecret),
	}, nil
}

// ReadCredentialsFromSecretRef reads the credentials from the secret referred by given <secretRef>.
func ReadCredentialsFromSecretRef(ctx context.Context, client client.Client, secretRef *corev1.SecretReference, allowDNSKeys bool) (*Credentials, error) {
	secret, err := extensionscontroller.GetSecretByReference(ctx, client, secretRef)
	if err != nil {
		return nil, err
	}

	return ReadSecretCredentials(secret, allowDNSKeys)
}

func getSecretDataValue(secret *corev1.Secret, key string, altKey *string) ([]byte, bool) {
	if value, ok := secret.Data[key]; ok {
		return value, true
	}
	if altKey != nil {
		if value, ok := secret.Data[*altKey]; ok {
			return value, true
		}
	}
	return nil, false
}
