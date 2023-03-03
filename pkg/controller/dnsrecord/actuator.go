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
	"strings"
	"time"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	"github.com/gardener/gardener/extensions/pkg/controller/dnsrecord"
	"github.com/gardener/gardener/extensions/pkg/util"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	extensionsv1alpha1helper "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1/helper"
	reconcilerutils "github.com/gardener/gardener/pkg/controllerutils/reconciler"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
)

const (
	// requeueAfterOnProviderError is a value for RequeueAfter to be returned on provider errors
	// in order to prevent quick retries that could quickly exhaust the account rate limits in case of e.g.
	// configuration issues.
	requeueAfterOnThrottlingError = 30 * time.Second
)

type actuator struct {
	common.ClientContext
	alicloudClientFactory alicloudclient.ClientFactory
}

// NewActuator creates a new dnsrecord.Actuator.
func NewActuator(alicloudClientFactory alicloudclient.ClientFactory, logger logr.Logger) dnsrecord.Actuator {
	return &actuator{
		alicloudClientFactory: alicloudClientFactory,
	}
}

// Reconcile reconciles the DNSRecord.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, dns *extensionsv1alpha1.DNSRecord, cluster *extensionscontroller.Cluster) error {
	// Create Alicloud client
	credentials, err := alicloud.ReadDNSCredentialsFromSecretRef(ctx, a.Client(), &dns.Spec.SecretRef)
	if err != nil {
		return fmt.Errorf("could not get Alicloud credentials: %+v", err)
	}
	dnsClient, err := a.alicloudClientFactory.NewDNSClient(getRegion(dns), string(credentials.AccessKeyID), string(credentials.AccessKeySecret))
	if err != nil {
		return util.DetermineError(fmt.Errorf("could not create Alicloud DNS client: %+v", err), helper.KnownCodes)
	}

	// Determine DNS domain name
	domainName, err := a.getDomainName(ctx, log, dns, dnsClient)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	// Create or update DNS records
	ttl := extensionsv1alpha1helper.GetDNSRecordTTL(dns.Spec.TTL)
	log.Info("Creating or updating DNS records", "domainName", domainName, "name", dns.Spec.Name, "type", dns.Spec.RecordType, "values", dns.Spec.Values, "dnsrecord", kutil.ObjectName(dns))
	if err := dnsClient.CreateOrUpdateDomainRecords(ctx, domainName, dns.Spec.Name, string(dns.Spec.RecordType), dns.Spec.Values, ttl); err != nil {
		return wrapAliClientError(err, fmt.Sprintf("could not create or update DNS records in domain %s with name %s, type %s, and values %v", domainName, dns.Spec.Name, dns.Spec.RecordType, dns.Spec.Values))
	}

	// Delete meta DNS records if any exist
	if dns.Status.LastOperation == nil || dns.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeCreate {
		name, recordType := dnsrecord.GetMetaRecordName(dns.Spec.Name), "TXT"
		log.Info("Deleting meta DNS records", "domainName", domainName, "name", name, "type", recordType, "dnsrecord", kutil.ObjectName(dns))
		if err := dnsClient.DeleteDomainRecords(ctx, domainName, name, recordType); err != nil {
			return wrapAliClientError(err, fmt.Sprintf("could not delete meta DNS records in domain %s with name %s and type %s", domainName, name, recordType))
		}
	}

	// Update resource status
	patch := client.MergeFrom(dns.DeepCopy())
	dns.Status.Zone = &domainName
	return a.Client().Status().Patch(ctx, dns, patch)
}

// Delete deletes the DNSRecord.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, dns *extensionsv1alpha1.DNSRecord, cluster *extensionscontroller.Cluster) error {
	// Create Alicloud client
	credentials, err := alicloud.ReadDNSCredentialsFromSecretRef(ctx, a.Client(), &dns.Spec.SecretRef)
	if err != nil {
		return fmt.Errorf("could not get Alicloud credentials: %+v", err)
	}
	dnsClient, err := a.alicloudClientFactory.NewDNSClient(getRegion(dns), string(credentials.AccessKeyID), string(credentials.AccessKeySecret))
	if err != nil {
		return util.DetermineError(fmt.Errorf("could not create Alicloud DNS client: %+v", err), helper.KnownCodes)
	}

	// Determine DNS domain name
	domainName, err := a.getDomainName(ctx, log, dns, dnsClient)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	// Delete DNS records
	log.Info("Deleting DNS records", "domainName", domainName, "name", dns.Spec.Name, "type", dns.Spec.RecordType, "dnsrecord", kutil.ObjectName(dns))
	if err := dnsClient.DeleteDomainRecords(ctx, domainName, dns.Spec.Name, string(dns.Spec.RecordType)); err != nil {
		return wrapAliClientError(err, fmt.Sprintf("could not delete DNS records in domain %s with name %s and type %s", domainName, dns.Spec.Name, dns.Spec.RecordType))
	}

	return nil
}

// Restore restores the DNSRecord.
func (a *actuator) Restore(ctx context.Context, log logr.Logger, dns *extensionsv1alpha1.DNSRecord, cluster *extensionscontroller.Cluster) error {
	return a.Reconcile(ctx, log, dns, cluster)
}

// Migrate migrates the DNSRecord.
func (a *actuator) Migrate(ctx context.Context, _ logr.Logger, dns *extensionsv1alpha1.DNSRecord, cluster *extensionscontroller.Cluster) error {
	return nil
}

func (a *actuator) getDomainName(ctx context.Context, log logr.Logger, dns *extensionsv1alpha1.DNSRecord, dnsClient alicloudclient.DNS) (string, error) {
	switch {
	case dns.Spec.Zone != nil && *dns.Spec.Zone != "" && (dns.Status.Zone == nil || *dns.Status.Zone == "" || !zoneMatchesDomainName(*dns.Spec.Zone, *dns.Status.Zone)):
		if isDomainName(*dns.Spec.Zone) {
			return *dns.Spec.Zone, nil
		}
		// The value specified in dns.Spec.Zone is not a domain name, so assume it's a domain id,
		// and try to determine the domain name by getting the name of the domain with this id
		domainName, err := dnsClient.GetDomainName(ctx, *dns.Spec.Zone)
		if err != nil {
			return "", wrapAliClientError(err, fmt.Sprintf("could not get DNS domain name for domain id %s", *dns.Spec.Zone))
		}
		log.Info("Got DNS domain name", "domainName", domainName, "dnsrecord", kutil.ObjectName(dns))
		return domainName, nil
	case dns.Status.Zone != nil && *dns.Status.Zone != "":
		return *dns.Status.Zone, nil
	default:
		// The zone is not specified in the resource status or spec. Try to determine the domain name by
		// getting all domain names of the account and searching for the longest domain name that is a suffix of dns.spec.Name
		domainNames, err := dnsClient.GetDomainNames(ctx)
		if err != nil {
			return "", wrapAliClientError(err, "could not get DNS domain names")
		}
		log.Info("Got DNS domain names", "domainNames", domainNames, "dnsrecord", kutil.ObjectName(dns))
		domainName := dnsrecord.FindZoneForName(domainNames, dns.Spec.Name)
		if domainName == "" {
			return "", fmt.Errorf("could not find DNS domain name for name %s", dns.Spec.Name)
		}
		return domainName, nil
	}
}

func zoneMatchesDomainName(zone, domainName string) bool {
	domainName, domainId := alicloudclient.DomainNameAndId(domainName)
	if isDomainName(zone) {
		return zone == domainName
	}
	return zone == domainId
}

// isDomainName returns true if the given zone contains at least one dot, false otherwise.
func isDomainName(zone string) bool {
	return strings.Contains(zone, ".")
}

func getRegion(dns *extensionsv1alpha1.DNSRecord) string {
	switch {
	case dns.Spec.Region != nil && *dns.Spec.Region != "":
		return *dns.Spec.Region
	default:
		return alicloud.DefaultDNSRegion
	}
}
func wrapAliClientError(err error, message string) error {
	wrappedErr := fmt.Errorf("%s: %+v", message, err)
	if _, ok := err.(*alicloudclient.RateLimiterWaitError); ok || alicloudclient.IsThrottlingError(err) {
		wrappedErr = &reconcilerutils.RequeueAfterError{
			Cause:        wrappedErr,
			RequeueAfter: requeueAfterOnThrottlingError,
		}
	}
	return wrappedErr
}
