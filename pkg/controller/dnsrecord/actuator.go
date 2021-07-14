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

package dnsrecord

import (
	"context"
	"fmt"
	"time"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	"github.com/gardener/gardener/extensions/pkg/controller/dnsrecord"
	controllererror "github.com/gardener/gardener/extensions/pkg/controller/error"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	extensionsv1alpha1helper "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1/helper"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/go-logr/logr"
	"k8s.io/client-go/util/retry"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
)

const (
	// requeueAfterOnProviderError is a value for RequeueAfter to be returned on provider errors
	// in order to prevent quick retries that could quickly exhaust the account rate limits in case of e.g.
	// configuration issues.
	requeueAfterOnProviderError = 30 * time.Second
)

type actuator struct {
	common.ClientContext
	alicloudClientFactory alicloudclient.ClientFactory
	logger                logr.Logger
}

func NewActuator(alicloudClientFactory alicloudclient.ClientFactory, logger logr.Logger) dnsrecord.Actuator {
	return &actuator{
		alicloudClientFactory: alicloudClientFactory,
		logger:                logger.WithName("alicloud-dnsrecord-actuator"),
	}
}

// Reconcile reconciles the DNSRecord.
func (a *actuator) Reconcile(ctx context.Context, dns *extensionsv1alpha1.DNSRecord, cluster *extensionscontroller.Cluster) error {
	// Create Alicloud client
	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, a.Client(), &dns.Spec.SecretRef, true)
	if err != nil {
		return fmt.Errorf("could not get Alicloud credentials: %+v", err)
	}
	dnsClient, err := a.alicloudClientFactory.NewDNSClient(getRegion(dns), string(credentials.AccessKeyID), string(credentials.AccessKeySecret))
	if err != nil {
		return fmt.Errorf("could not create Alicloud DNS client: %+v", err)
	}

	// Determine DNS domain name
	domainName, err := a.getDomainName(ctx, dns, dnsClient)
	if err != nil {
		return err
	}

	// Create or update DNS records
	ttl := extensionsv1alpha1helper.GetDNSRecordTTL(dns.Spec.TTL)
	a.logger.Info("Creating or updating DNS records", "domainName", domainName, "name", dns.Spec.Name, "type", dns.Spec.RecordType, "values", dns.Spec.Values, "dnsrecord", kutil.ObjectName(dns))
	if err := dnsClient.CreateOrUpdateDomainRecords(ctx, domainName, dns.Spec.Name, string(dns.Spec.RecordType), dns.Spec.Values, ttl); err != nil {
		return &controllererror.RequeueAfterError{
			Cause:        fmt.Errorf("could not create or update DNS records in domain %s with name %s, type %s, and values %v: %+v", domainName, dns.Spec.Name, dns.Spec.RecordType, dns.Spec.Values, err),
			RequeueAfter: requeueAfterOnProviderError,
		}
	}

	// Delete meta DNS records if any exist
	if dns.Status.LastOperation == nil || dns.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeCreate {
		name, recordType := dnsrecord.GetMetaRecordName(dns.Spec.Name), "TXT"
		a.logger.Info("Deleting meta DNS records", "domainName", domainName, "name", name, "type", recordType, "dnsrecord", kutil.ObjectName(dns))
		if err := dnsClient.DeleteDomainRecords(ctx, domainName, name, recordType); err != nil {
			return &controllererror.RequeueAfterError{
				Cause:        fmt.Errorf("could not delete meta DNS records in domain %s with name %s and type %s: %+v", domainName, name, recordType, err),
				RequeueAfter: requeueAfterOnProviderError,
			}
		}
	}

	// Update resource status
	return extensionscontroller.TryUpdateStatus(ctx, retry.DefaultBackoff, a.Client(), dns, func() error {
		dns.Status.Zone = &domainName
		return nil
	})
}

// Delete deletes the DNSRecord.
func (a *actuator) Delete(ctx context.Context, dns *extensionsv1alpha1.DNSRecord, cluster *extensionscontroller.Cluster) error {
	// Create Alicloud client
	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, a.Client(), &dns.Spec.SecretRef, true)
	if err != nil {
		return fmt.Errorf("could not get Alicloud credentials: %+v", err)
	}
	dnsClient, err := a.alicloudClientFactory.NewDNSClient(getRegion(dns), string(credentials.AccessKeyID), string(credentials.AccessKeySecret))
	if err != nil {
		return fmt.Errorf("could not create Alicloud DNS client: %+v", err)
	}

	// Determine DNS domain name
	domainName, err := a.getDomainName(ctx, dns, dnsClient)
	if err != nil {
		return err
	}

	// Delete DNS records
	a.logger.Info("Deleting DNS records", "domainName", domainName, "name", dns.Spec.Name, "type", dns.Spec.RecordType, "dnsrecord", kutil.ObjectName(dns))
	if err := dnsClient.DeleteDomainRecords(ctx, domainName, dns.Spec.Name, string(dns.Spec.RecordType)); err != nil {
		return &controllererror.RequeueAfterError{
			Cause:        fmt.Errorf("could not delete DNS records in domain %s with name %s and type %s: %+v", domainName, dns.Spec.Name, dns.Spec.RecordType, err),
			RequeueAfter: requeueAfterOnProviderError,
		}
	}

	return nil
}

// Restore restores the DNSRecord.
func (a *actuator) Restore(ctx context.Context, dns *extensionsv1alpha1.DNSRecord, cluster *extensionscontroller.Cluster) error {
	return a.Reconcile(ctx, dns, cluster)
}

// Migrate migrates the DNSRecord.
func (a *actuator) Migrate(ctx context.Context, dns *extensionsv1alpha1.DNSRecord, cluster *extensionscontroller.Cluster) error {
	return nil
}

func (a *actuator) getDomainName(ctx context.Context, dns *extensionsv1alpha1.DNSRecord, dnsClient alicloudclient.DNS) (string, error) {
	switch {
	case dns.Spec.Zone != nil && *dns.Spec.Zone != "":
		return *dns.Spec.Zone, nil
	case dns.Status.Zone != nil && *dns.Status.Zone != "":
		return *dns.Status.Zone, nil
	default:
		// The zone is not specified in the resource status or spec. Try to determine the domain name by
		// getting all domain names of the account and searching for the longest domain name that is a suffix of dns.spec.Name
		domainNames, err := dnsClient.GetDomainNames(ctx)
		if err != nil {
			return "", &controllererror.RequeueAfterError{
				Cause:        fmt.Errorf("could not get DNS domain names: %+v", err),
				RequeueAfter: requeueAfterOnProviderError,
			}
		}
		a.logger.Info("Got DNS domain names", "domainNames", domainNames, "dnsrecord", kutil.ObjectName(dns))
		domainName := dnsrecord.FindZoneForName(toMap(domainNames), dns.Spec.Name)
		if domainName == "" {
			return "", fmt.Errorf("could not find DNS domain name for name %s", dns.Spec.Name)
		}
		return domainName, nil
	}
}

func getRegion(dns *extensionsv1alpha1.DNSRecord) string {
	switch {
	case dns.Spec.Region != nil && *dns.Spec.Region != "":
		return *dns.Spec.Region
	default:
		return alicloud.DefaultDNSRegion
	}
}

func toMap(ss []string) map[string]string {
	m := make(map[string]string)
	for _, s := range ss {
		m[s] = s
	}
	return m
}
