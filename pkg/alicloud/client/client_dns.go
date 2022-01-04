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

package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	"golang.org/x/time/rate"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/gardener/gardener/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	domainsCacheTTL     = 1 * time.Hour
	rateLimiterCacheTTL = 1 * time.Hour
)

// RateLimiterWaitError is an error to be reported if waiting for a aliyun dns  rate limiter fails.
// This can only happen if the wait time would exceed the configured wait timeout.
type RateLimiterWaitError struct {
	Cause error
}

func (e *RateLimiterWaitError) Error() string {
	return fmt.Sprintf("could not wait for client-side route53 rate limiter: %+v", e.Cause)
}

// NewDNSClient creates a new DNS client with given region, accessKeyID, and accessKeySecret.
func (f *clientFactory) NewDNSClient(region, accessKeyID, accessKeySecret string) (DNS, error) {
	client, err := alidns.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}

	return &dnsClient{
		Client:                 *client,
		accessKeyID:            accessKeyID,
		domainsCache:           f.domainsCache,
		domainsCacheMutex:      &f.domainsCacheMutex,
		RateLimiter:            f.getRateLimiter(accessKeyID),
		RateLimiterWaitTimeout: f.waitTimeout,
		Logger:                 log.Log.WithName("ali-dnsclient"),
	}, nil
}
func (f *clientFactory) getRateLimiter(accessKeyID string) *rate.Limiter {
	// cache.Expiring Get and Set methods are concurrency-safe
	// However, if f rate limiter is not present in the cache, it may happen that multiple rate limiters are created
	// at the same time for the same access key id, and the desired QPS is exceeded, so use f mutex to guard against this

	f.rateLimitersMutex.Lock()
	defer f.rateLimitersMutex.Unlock()

	// Get f rate limiter from the cache, or create f new one if not present
	var rateLimiter *rate.Limiter
	if v, ok := f.rateLimiters.Get(accessKeyID); ok {
		rateLimiter = v.(*rate.Limiter)
	} else {
		rateLimiter = rate.NewLimiter(f.limit, f.burst)
	}
	// Set should be called on every Get with cache.Expiring to refresh the TTL
	f.rateLimiters.Set(accessKeyID, rateLimiter, rateLimiterCacheTTL)
	return rateLimiter
}

// GetDomainNames returns a map of all domain names mapped to their composite domain names.
func (d *dnsClient) GetDomainNames(ctx context.Context) (map[string]string, error) {
	domains, err := d.getDomainsWithCache(ctx)
	if err != nil {
		return nil, err
	}
	domainNames := make(map[string]string)
	for _, domain := range domains {
		domainNames[domain.DomainName] = CompositeDomainName(domain.DomainName, domain.DomainId)
	}
	return domainNames, nil
}

// GetDomainName returns the composite domain name of the domain with the given domain id.
func (d *dnsClient) GetDomainName(ctx context.Context, domainId string) (string, error) {
	domains, err := d.getDomainsWithCache(ctx)
	if err != nil {
		return "", err
	}
	domain, ok := domains[domainId]
	if !ok {
		return "", fmt.Errorf("DNS domain with id %s not found", domainId)
	}
	return CompositeDomainName(domain.DomainName, domain.DomainId), nil
}

// CreateOrUpdateDomainRecords creates or updates the domain records with the given domain name, name, record type,
// values, and ttl.
// * For each element in values that has an existing domain record, the existing record is updated if needed.
// * For each element in values that doesn't have an existing domain record, a new domain record is created.
// * For each existing domain record that doesn't have a corresponding element in values, the existing record is deleted.
func (d *dnsClient) CreateOrUpdateDomainRecords(ctx context.Context, domainName, name, recordType string, values []string, ttl int64) error {
	domainName, _ = DomainNameAndId(domainName)
	rr, err := getRR(name, domainName)
	if err != nil {
		return err
	}
	records, err := d.getDomainRecords(ctx, domainName, rr, recordType)
	if err != nil {
		return err
	}
	for _, value := range values {
		if record, ok := records[value]; ok {
			// Only update the existing domain record if the current TTL value is different from the given one
			// At this point we know that rr, recordType, and value are the same
			if record.TTL != ttl {
				if err := d.updateDomainRecord(ctx, record.RecordId, rr, recordType, value, ttl); err != nil {
					return err
				}
			}
		} else {
			if err := d.createDomainRecord(ctx, domainName, rr, recordType, value, ttl); err != nil {
				return err
			}
		}
	}
	for value, record := range records {
		if !utils.ValueExists(value, values) {
			if err := d.deleteDomainRecord(ctx, record.RecordId); err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteDomainRecords deletes the domain records with the given domain name, name and record type.
func (d *dnsClient) DeleteDomainRecords(ctx context.Context, domainName, name, recordType string) error {
	domainName, _ = DomainNameAndId(domainName)
	rr, err := getRR(name, domainName)
	if err != nil {
		return err
	}
	records, err := d.getDomainRecords(ctx, domainName, rr, recordType)
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := d.deleteDomainRecord(ctx, record.RecordId); err != nil {
			return err
		}
	}
	return nil
}

func (d *dnsClient) getDomainsWithCache(ctx context.Context) (map[string]alidns.Domain, error) {
	// cache.Expiring Get and Set methods are concurrency-safe.
	// However, if an accessKeyID is not present in the cache and multiple DNSRecords are reconciled at the same time,
	// it may happen that getDomains is called multiple times instead of just one, so use a mutex to guard against this.
	// It is ok to use a shared mutex here as far as the number of accessKeyIDs using custom domains is low.
	// This may need to be revisited with a larger number of such accessKeyIDs to avoid them blocking each other
	// during the (potentially long-running) call to getDomains.
	d.domainsCacheMutex.Lock()
	defer d.domainsCacheMutex.Unlock()

	if v, ok := d.domainsCache.Get(d.accessKeyID); ok {
		return v.(map[string]alidns.Domain), nil
	}
	domains, err := d.getDomains(ctx)
	if err != nil {
		return nil, err
	}
	d.domainsCache.Set(d.accessKeyID, domains, domainsCacheTTL)
	return domains, nil
}

// getDomains returns all domains.
func (d *dnsClient) getDomains(ctx context.Context) (map[string]alidns.Domain, error) {
	if err := d.waitForRoute53RateLimiter(ctx); err != nil {
		return nil, err
	}

	domains := make(map[string]alidns.Domain)
	pageSize, pageNumber := 20, 1
	req := alidns.CreateDescribeDomainsRequest()
	req.PageSize = requests.NewInteger(pageSize)
	for {
		req.PageNumber = requests.NewInteger(pageNumber)
		resp, err := d.Client.DescribeDomains(req)
		if err != nil {
			return nil, err
		}
		for _, domain := range resp.Domains.Domain {
			domains[domain.DomainId] = domain
		}
		if resp.PageNumber*int64(pageSize) >= resp.TotalCount {
			break
		}
		pageNumber++
	}
	return domains, nil
}

// getDomainRecords returns the domain records with the given domain name, rr, and record type.
func (d *dnsClient) getDomainRecords(ctx context.Context, domainName, rr, recordType string) (map[string]alidns.Record, error) {
	if err := d.waitForRoute53RateLimiter(ctx); err != nil {
		return nil, err
	}

	records := make(map[string]alidns.Record)
	pageSize, pageNumber := 20, 1
	req := alidns.CreateDescribeDomainRecordsRequest()
	req.PageSize = requests.NewInteger(pageSize)
	for {
		req.PageNumber = requests.NewInteger(pageNumber)
		req.DomainName = domainName
		req.RRKeyWord = rr
		req.TypeKeyWord = recordType
		resp, err := d.Client.DescribeDomainRecords(req)
		if err != nil {
			return nil, err
		}
		for _, record := range resp.DomainRecords.Record {
			records[record.Value] = record
		}
		if resp.PageNumber*int64(pageSize) >= resp.TotalCount {
			break
		}
		pageNumber++
	}
	return records, nil
}

func (d *dnsClient) createDomainRecord(ctx context.Context, domainName, rr, recordType, value string, ttl int64) error {
	if err := d.waitForRoute53RateLimiter(ctx); err != nil {
		return err
	}

	req := alidns.CreateAddDomainRecordRequest()
	req.DomainName = domainName
	req.RR = rr
	req.Type = recordType
	req.Value = value
	req.TTL = requests.NewInteger(int(ttl))
	_, err := d.Client.AddDomainRecord(req)
	return err
}

func (d *dnsClient) updateDomainRecord(ctx context.Context, id, rr, recordType, value string, ttl int64) error {
	if err := d.waitForRoute53RateLimiter(ctx); err != nil {
		return err
	}

	req := alidns.CreateUpdateDomainRecordRequest()
	req.RecordId = id
	req.RR = rr
	req.Type = recordType
	req.Value = value
	req.TTL = requests.NewInteger(int(ttl))
	_, err := d.Client.UpdateDomainRecord(req)
	return err
}

func (d *dnsClient) deleteDomainRecord(ctx context.Context, id string) error {
	if err := d.waitForRoute53RateLimiter(ctx); err != nil {
		return err
	}

	req := alidns.CreateDeleteDomainRecordRequest()
	req.RecordId = id
	if _, err := d.Client.DeleteDomainRecord(req); err != nil && !isDomainRecordDoesNotExistError(err) {
		return err
	}
	return nil
}

func (c *dnsClient) waitForRoute53RateLimiter(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.RateLimiterWaitTimeout)
	defer cancel()
	t := time.Now()
	if err := c.RateLimiter.Wait(timeoutCtx); err != nil {
		return &RateLimiterWaitError{Cause: err}
	}
	if waitDuration := time.Since(t); waitDuration.Seconds() > 1/float64(c.RateLimiter.Limit()) {
		c.Logger.Info("Waited for client-side aliyun DNS rate limiter", "waitDuration", waitDuration.String())
	}
	return nil
}
func getRR(name, domainName string) (string, error) {
	if name == domainName {
		return "@", nil
	}
	suffix := "." + domainName
	if !strings.HasSuffix(name, suffix) {
		return "", fmt.Errorf("name %s does not match domain name %s", name, domainName)
	}
	return strings.TrimSuffix(name, suffix), nil
}

func isDomainRecordDoesNotExistError(err error) bool {
	if serverError, ok := err.(*errors.ServerError); ok {
		if serverError.ErrorCode() == alicloud.ErrorCodeDomainRecordNotBelongToUser {
			return true
		}
	}
	return false
}

// CompositeDomainName composes and returns a composite domain name from the given domain name and id,
// in the format <domainName>:<domainId>
func CompositeDomainName(domainName, domainId string) string {
	if domainId != "" {
		return domainName + ":" + domainId
	}
	return domainName
}

// DomainNameAndId decomposes the given composite domain name in the format <domainName>:<domainId>
// into its constituent domain name and id.
func DomainNameAndId(compositeDomainName string) (string, string) {
	if parts := strings.Split(compositeDomainName, ":"); len(parts) == 2 {
		return parts[0], parts[1]
	}
	return compositeDomainName, ""
}

// IsThrottlingError returns true if the error is a throttling error.
func IsThrottlingError(err error) bool {
	if alierr, ok := err.(errors.Error); ok && strings.Contains(alierr.Message(), "Throttling") {
		return true
	}
	return false
}
