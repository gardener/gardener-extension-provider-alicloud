// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	"context"
	"fmt"
	"time"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	securityv1alpha1constants "github.com/gardener/gardener/pkg/apis/security/v1alpha1/constants"
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

// AuthConfig holds credentials for either static AK/SK or workload identity mode.
// Exactly one of AccessKey or WorkloadIdentity is set.
// CredentialsFile is populated by the cloudprovider webhook in AK mode only.
type AuthConfig struct {
	AccessKey        *AccessKey
	WorkloadIdentity *WorkloadIdentityAuth
	CredentialsFile  string
}

// AccessKey holds static Alicloud access key credentials.
type AccessKey struct {
	ID     string
	Secret string
}

// WorkloadIdentityAuth holds the data needed to exchange a JWT token for temporary STS credentials via RRSA.
type WorkloadIdentityAuth struct {
	OIDCProviderARN string
	RoleARN         string
	Token           []byte
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

	accessKeyID, ok := getSecretDataValue(secret, AccessKeyID, altAccessKeyIDKey)
	if !ok {
		return nil, fmt.Errorf("secret %s/%s has no access key id", secret.Namespace, secret.Name)
	}

	accessKeySecret, ok := getSecretDataValue(secret, AccessKeySecret, altAccessKeySecretKey)
	if !ok {
		return nil, fmt.Errorf("secret %s/%s has no access key secret", secret.Namespace, secret.Name)
	}

	credentialsFile, ok := getSecretDataValue(secret, CredentialsFile, nil)
	if ok {
		return &Credentials{
			AccessKeyID:     string(accessKeyID),
			AccessKeySecret: string(accessKeySecret),
			CredentialsFile: string(credentialsFile),
		}, nil
	}

	return &Credentials{
		AccessKeyID:     string(accessKeyID),
		AccessKeySecret: string(accessKeySecret),
	}, nil
}

// ReadDNSAuthConfigFromSecretRef reads an AuthConfig from the DNS secret referred by the given secretRef.
// It accepts both standard keys (accessKeyID/accessKeySecret) and DNS-specific keys (ACCESS_KEY_ID/ACCESS_KEY_SECRET).
func ReadDNSAuthConfigFromSecretRef(ctx context.Context, c client.Client, secretRef *corev1.SecretReference) (*AuthConfig, error) {
	secret, err := extensionscontroller.GetSecretByReference(ctx, c, secretRef)
	if err != nil {
		return nil, err
	}
	return ReadAuthConfig(secret, true)
}

// ReadAuthConfigFromSecretRef reads an AuthConfig from the secret referred by the given secretRef.
func ReadAuthConfigFromSecretRef(ctx context.Context, c client.Client, secretRef *corev1.SecretReference) (*AuthConfig, error) {
	secret, err := extensionscontroller.GetSecretByReference(ctx, c, secretRef)
	if err != nil {
		return nil, err
	}
	return ReadAuthConfig(secret, false)
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

// ReadAuthConfig reads an AuthConfig from the given secret.
// If allowDNSKeys is true, it also accepts DNS-specific key names (ACCESS_KEY_ID/ACCESS_KEY_SECRET).
// If the secret contains accessKeyID it returns AK/SK mode; otherwise it expects a workload identity token.
func ReadAuthConfig(secret *corev1.Secret, allowDNSKeys bool) (*AuthConfig, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret %s/%s has no data section", secret.Namespace, secret.Name)
	}

	var altAKIDKey, altAKSecretKey *string
	if allowDNSKeys {
		altAKIDKey = ptr.To(dnsAccessKeyID)
		altAKSecretKey = ptr.To(dnsAccessKeySecret)
	}

	if akID, ok := getSecretDataValue(secret, AccessKeyID, altAKIDKey); ok {
		akSecret, ok := getSecretDataValue(secret, AccessKeySecret, altAKSecretKey)
		if !ok {
			return nil, fmt.Errorf("secret %s/%s has accessKeyID but no accessKeySecret", secret.Namespace, secret.Name)
		}
		return &AuthConfig{
			AccessKey:       &AccessKey{ID: string(akID), Secret: string(akSecret)},
			CredentialsFile: string(secret.Data[CredentialsFile]),
		}, nil
	}

	token, ok := secret.Data[securityv1alpha1constants.DataKeyToken]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s has neither accessKeyID nor workload identity token", secret.Namespace, secret.Name)
	}
	return &AuthConfig{
		WorkloadIdentity: &WorkloadIdentityAuth{
			OIDCProviderARN: string(secret.Data[OIDCProviderARNKey]),
			RoleARN:         string(secret.Data[RoleARNKey]),
			Token:           token,
		},
		CredentialsFile: string(secret.Data[CredentialsFile]),
	}, nil
}

// BuildSessionName constructs an STS RoleSessionName from a Kubernetes namespace.
// STS allows 2-64 characters (letters, digits, -, _, ., @).
// The namespace is truncated to 44 chars so that the gardener- prefix and Unix
// timestamp suffix keep the total within the 64-char limit.
func BuildSessionName(namespace string) string {
	ns := namespace
	if len(ns) > 44 {
		ns = ns[:44]
	}
	return fmt.Sprintf("gardener-%s-%d", ns, time.Now().Unix())
}
