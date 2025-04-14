// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Credentials are the credentials to access the Alicloud API.
type Credentials struct {
	AccessKeyID     string
	AccessKeySecret string
	CredentialsFile string
}

const (
	// AccessKeyID is the data field in a secret where the access key id is stored at.
	AccessKeyID = "accessKeyID"
	// AccessKeySecret is the data field in a secret where the access key secret is stored at.
	AccessKeySecret = "accessKeySecret"
	// CredentialsFile is a constant for the key in cloud provider secret that holds the Alibaba Cloud credentials file.
	CredentialsFile = "credentialsFile"

	// dnsAccessKeyID is the data field in a DNS secret where the access key id is stored at.
	dnsAccessKeyID = "ACCESS_KEY_ID"
	// DNSAccessKeySecret is the data field in a DNS secret where the access key secret is stored at.
	dnsAccessKeySecret = "ACCESS_KEY_SECRET" // #nosec: G101
)

// ReadSecretCredentials reads the Credentials from the given secret.
func ReadSecretCredentials(secret *corev1.Secret, allowDNSKeys bool) (*Credentials, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret %s/%s has no data section", secret.Namespace, secret.Name)
	}

	var altAccessKeyIDKey, altAccessKeySecretKey *string
	if allowDNSKeys {
		altAccessKeyIDKey, altAccessKeySecretKey = ptr.To(dnsAccessKeyID), ptr.To(dnsAccessKeySecret)
	}

	accessKeyID, ok := getSecretDataValue(secret, AccessKeyID, altAccessKeyIDKey, true)
	if !ok {
		return nil, fmt.Errorf("secret %s/%s has no access key id", secret.Namespace, secret.Name)
	}

	accessKeySecret, ok := getSecretDataValue(secret, AccessKeySecret, altAccessKeySecretKey, true)
	if !ok {
		return nil, fmt.Errorf("secret %s/%s has no access key secret", secret.Namespace, secret.Name)
	}

	credentialsFile, _ := getSecretDataValue(secret, CredentialsFile, nil, false)

	return &Credentials{
		AccessKeyID:     string(accessKeyID),
		AccessKeySecret: string(accessKeySecret),
		CredentialsFile: string(credentialsFile),
	}, nil
}

// ReadCredentialsFromSecretRef reads the credentials from the secret referred by given <secretRef>.
func ReadCredentialsFromSecretRef(ctx context.Context, client client.Client, secretRef *corev1.SecretReference) (*Credentials, error) {
	secret, err := extensionscontroller.GetSecretByReference(ctx, client, secretRef)
	if err != nil {
		return nil, err
	}

	return ReadSecretCredentials(secret, false)
}

// ReadDNSCredentialsFromSecretRef reads the credentials from the DNS secret referred by given <secretRef>.
func ReadDNSCredentialsFromSecretRef(ctx context.Context, client client.Client, secretRef *corev1.SecretReference) (*Credentials, error) {
	secret, err := extensionscontroller.GetSecretByReference(ctx, client, secretRef)
	if err != nil {
		return nil, err
	}

	return ReadSecretCredentials(secret, true)
}

func getSecretDataValue(secret *corev1.Secret, key string, altKey *string, required bool) ([]byte, bool) {
	if value, ok := secret.Data[key]; ok {
		return value, true
	}
	if altKey != nil {
		if value, ok := secret.Data[*altKey]; ok {
			return value, true
		}
	}
	if required {
		return nil, false
	}
	return nil, true
}
