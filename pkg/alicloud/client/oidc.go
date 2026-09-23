// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"os"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
)

// NewOIDCCredentialsProvider writes the given JWT token to a temporary file and builds
// an OIDCCredentialsProvider that will exchange it for short-lived STS credentials via RRSA.
// sessionName is used as RoleSessionName in the STS AssumeRoleWithOIDC call and appears in
// ActionTrail audit logs; callers should embed shoot identity and a timestamp for traceability.
// The returned cleanup function removes the temporary file and must be called when the
// provider is no longer needed (typically via defer).
func NewOIDCCredentialsProvider(tokenBytes []byte, oidcProviderARN, roleARN, sessionName string) (
	*credentials.OIDCCredentialsProvider, func(), error,
) {
	f, err := os.CreateTemp("", "gardener-wi-token-*.jwt")
	if err != nil {
		return nil, nil, fmt.Errorf("creating temporary OIDC token file: %w", err)
	}

	if _, err := f.Write(tokenBytes); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return nil, nil, fmt.Errorf("writing OIDC token to temporary file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return nil, nil, fmt.Errorf("closing temporary OIDC token file: %w", err)
	}

	provider, err := credentials.NewOIDCCredentialsProviderBuilder().
		WithOIDCProviderARN(oidcProviderARN).
		WithRoleArn(roleARN).
		WithOIDCTokenFilePath(f.Name()).
		WithRoleSessionName(sessionName).
		Build()
	if err != nil {
		_ = os.Remove(f.Name())
		return nil, nil, fmt.Errorf("building OIDC credentials provider: %w", err)
	}

	cleanup := func() { _ = os.Remove(f.Name()) }
	return provider, cleanup, nil
}
