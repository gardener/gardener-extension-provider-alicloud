// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/nlb"
	ram "github.com/aliyun/alibaba-cloud-sdk-go/services/resourcemanager"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client/ros"
)

// exchangeOIDCForSTS writes the JWT to a temp file, calls STS AssumeRoleWithOIDC, returns the
// resulting short-lived credentials, and cleans up the temp file before returning.
func exchangeOIDCForSTS(auth alicloud.AuthConfig, sessionName string) (ak, sk, token string, err error) {
	wi := auth.WorkloadIdentity
	provider, cleanup, err := NewOIDCCredentialsProvider(wi.Token, wi.OIDCProviderARN, wi.RoleARN, sessionName)
	if err != nil {
		return "", "", "", err
	}
	defer cleanup()

	creds, err := provider.GetCredentials()
	if err != nil {
		return "", "", "", fmt.Errorf("exchanging OIDC token for STS credentials: %w", err)
	}
	return creds.AccessKeyId, creds.AccessKeySecret, creds.SecurityToken, nil
}

// NewECSClientFromAuthConfig creates an ECS client from an AuthConfig (static AK or workload identity).
func (f *clientFactory) NewECSClientFromAuthConfig(region string, auth alicloud.AuthConfig, sessionName string) (ECS, error) {
	if auth.AccessKey != nil {
		return f.NewECSClient(region, auth.AccessKey.ID, auth.AccessKey.Secret)
	}
	ak, sk, token, err := exchangeOIDCForSTS(auth, sessionName)
	if err != nil {
		return nil, err
	}
	c, err := ecs.NewClientWithStsToken(region, ak, sk, token)
	if err != nil {
		return nil, err
	}
	return &ecsClient{*c}, nil
}

// NewSTSClientFromAuthConfig creates an STS client from an AuthConfig (static AK or workload identity).
func (f *clientFactory) NewSTSClientFromAuthConfig(region string, auth alicloud.AuthConfig, sessionName string) (STS, error) {
	if auth.AccessKey != nil {
		return f.NewSTSClient(region, auth.AccessKey.ID, auth.AccessKey.Secret)
	}
	ak, sk, token, err := exchangeOIDCForSTS(auth, sessionName)
	if err != nil {
		return nil, err
	}
	c, err := sts.NewClientWithStsToken(region, ak, sk, token)
	if err != nil {
		return nil, err
	}
	return &stsClient{*c}, nil
}

// NewSLBClientFromAuthConfig creates an SLB client from an AuthConfig (static AK or workload identity).
func (f *clientFactory) NewSLBClientFromAuthConfig(region string, auth alicloud.AuthConfig, sessionName string) (SLB, error) {
	if auth.AccessKey != nil {
		return f.NewSLBClient(region, auth.AccessKey.ID, auth.AccessKey.Secret)
	}
	ak, sk, token, err := exchangeOIDCForSTS(auth, sessionName)
	if err != nil {
		return nil, err
	}
	c, err := slb.NewClientWithStsToken(region, ak, sk, token)
	if err != nil {
		return nil, err
	}
	return &slbClient{*c}, nil
}

// NewVPCClientFromAuthConfig creates a VPC client from an AuthConfig (static AK or workload identity).
func (f *clientFactory) NewVPCClientFromAuthConfig(region string, auth alicloud.AuthConfig, sessionName string) (VPC, error) {
	if auth.AccessKey != nil {
		return f.NewVPCClient(region, auth.AccessKey.ID, auth.AccessKey.Secret)
	}
	ak, sk, token, err := exchangeOIDCForSTS(auth, sessionName)
	if err != nil {
		return nil, err
	}
	c, err := vpc.NewClientWithStsToken(region, ak, sk, token)
	if err != nil {
		return nil, err
	}
	return &vpcClient{*c}, nil
}

// NewRAMClientFromAuthConfig creates a RAM client from an AuthConfig (static AK or workload identity).
func (f *clientFactory) NewRAMClientFromAuthConfig(region string, auth alicloud.AuthConfig, sessionName string) (RAM, error) {
	if auth.AccessKey != nil {
		return f.NewRAMClient(region, auth.AccessKey.ID, auth.AccessKey.Secret)
	}
	ak, sk, token, err := exchangeOIDCForSTS(auth, sessionName)
	if err != nil {
		return nil, err
	}
	c, err := ram.NewClientWithStsToken(region, ak, sk, token)
	if err != nil {
		return nil, err
	}
	return &ramClient{*c}, nil
}

// NewROSClientFromAuthConfig creates a ROS client from an AuthConfig (static AK or workload identity).
func (f *clientFactory) NewROSClientFromAuthConfig(region string, auth alicloud.AuthConfig, sessionName string) (ROS, error) {
	if auth.AccessKey != nil {
		return f.NewROSClient(region, auth.AccessKey.ID, auth.AccessKey.Secret)
	}
	ak, sk, token, err := exchangeOIDCForSTS(auth, sessionName)
	if err != nil {
		return nil, err
	}
	c := &ros.Client{}
	if err := c.InitWithStsToken(region, ak, sk, token); err != nil {
		return nil, err
	}
	ros.SetEndpointDataToClient(c)
	return c, nil
}

// NewOSSClientFromAuthConfig creates an OSS client from an AuthConfig (static AK or workload identity).
func (f *clientFactory) NewOSSClientFromAuthConfig(endpoint string, auth alicloud.AuthConfig, sessionName string) (OSS, error) {
	if auth.AccessKey != nil {
		return f.NewOSSClient(endpoint, auth.AccessKey.ID, auth.AccessKey.Secret)
	}
	ak, sk, token, err := exchangeOIDCForSTS(auth, sessionName)
	if err != nil {
		return nil, err
	}
	c, err := oss.New(endpoint, ak, sk, oss.SecurityToken(token))
	if err != nil {
		return nil, err
	}
	return &ossClient{*c}, nil
}

// NewDNSClientFromAuthConfig creates a DNS client from an AuthConfig (static AK or workload identity).
func (f *clientFactory) NewDNSClientFromAuthConfig(region string, auth alicloud.AuthConfig, sessionName string) (DNS, error) {
	if auth.AccessKey != nil {
		return f.NewDNSClient(region, auth.AccessKey.ID, auth.AccessKey.Secret)
	}
	ak, sk, token, err := exchangeOIDCForSTS(auth, sessionName)
	if err != nil {
		return nil, err
	}
	c, err := alidns.NewClientWithStsToken(region, ak, sk, token)
	if err != nil {
		return nil, err
	}
	c.AppendUserAgent("application", "gardener-extension-provider-alicloud")
	return &dnsClient{
		Client:                 *c,
		accessKeyID:            ak,
		domainsCache:           f.domainsCache,
		domainsCacheMutex:      &f.domainsCacheMutex,
		RateLimiter:            f.getRateLimiter(ak),
		RateLimiterWaitTimeout: f.waitTimeout,
		Logger:                 log.Log.WithName("ali-dnsclient"),
	}, nil
}

// NewNLBClientFromAuthConfig creates an NLB client from an AuthConfig (static AK or workload identity).
func (f *clientFactory) NewNLBClientFromAuthConfig(region string, auth alicloud.AuthConfig, sessionName string) (NLB, error) {
	if auth.AccessKey != nil {
		return f.NewNLBClient(region, auth.AccessKey.ID, auth.AccessKey.Secret)
	}
	ak, sk, token, err := exchangeOIDCForSTS(auth, sessionName)
	if err != nil {
		return nil, err
	}
	c, err := nlb.NewClientWithStsToken(region, ak, sk, token)
	if err != nil {
		return nil, err
	}
	return &nlbClient{*c}, nil
}
