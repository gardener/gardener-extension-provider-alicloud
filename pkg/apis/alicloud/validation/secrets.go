// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
)

const (
	accessKeyIDMinLen     = 16
	accessKeyIDMaxLen     = 128
	accessKeySecretMinLen = 30
)

var (
	// accessKeyID accepts only alphanumeric characters [0-9a-zA-Z] and several special characters [._=],
	// see https://partners-intl.aliyun.com/help/doc-detail/185803.htm
	accessKeyIDRegex = regexp.MustCompile(`^[0-9a-zA-Z._=]+$`)
)

// ValidateCloudProviderSecret checks whether the given secret contains a valid Alicloud access keys.
func ValidateCloudProviderSecret(secret *corev1.Secret) error {
	secretRef := fmt.Sprintf("%s/%s", secret.Namespace, secret.Name)

	accessKeyID, ok := secret.Data[alicloud.AccessKeyID]
	if !ok {
		return fmt.Errorf("missing %q field in secret %s", alicloud.AccessKeyID, secretRef)
	}
	if len(accessKeyID) < accessKeyIDMinLen {
		return fmt.Errorf("field %q in secret %s must have at least %d characters", alicloud.AccessKeyID, secretRef, accessKeyIDMinLen)
	}
	if len(accessKeyID) > accessKeyIDMaxLen {
		return fmt.Errorf("field %q in secret %s cannot be longer than %d characters", alicloud.AccessKeyID, secretRef, accessKeyIDMaxLen)
	}
	if !accessKeyIDRegex.Match(accessKeyID) {
		return fmt.Errorf("field %q in secret %s must only contain alphanumeric characters and [._=]", alicloud.AccessKeyID, secretRef)
	}

	secretAccessKey, ok := secret.Data[alicloud.AccessKeySecret]
	if !ok {
		return fmt.Errorf("missing %q field in secret %s", alicloud.AccessKeySecret, secretRef)
	}
	if len(secretAccessKey) < accessKeySecretMinLen {
		return fmt.Errorf("field %q in secret %s must have at least %d characters", alicloud.AccessKeySecret, secretRef, accessKeySecretMinLen)
	}
	// accessKeySecret must not contain leading or trailing new lines, as they are known to cause issues
	// Other whitespace characters such as spaces are intentionally not checked for,
	// since there is no documentation indicating that they would not be valid
	if strings.Trim(string(secretAccessKey), "\n\r") != string(secretAccessKey) {
		return fmt.Errorf("field %q in secret %s must not contain leading or traling new lines", alicloud.AccessKeySecret, secretRef)
	}

	return nil
}
