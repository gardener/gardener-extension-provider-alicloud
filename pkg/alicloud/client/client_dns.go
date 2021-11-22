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

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/gardener/gardener/pkg/utils"
)

const (
	domainsCacheTTL = 1 * time.Hour
)

// NewDNSClient creates a new DNS client with given region, accessKeyID, and accessKeySecret.
func (f *clientFactory) NewDNSClient(region, accessKeyID, accessKeySecret string) (DNS, error) {
	client, err := alidns.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}

	return &dnsClient{
		Client:            *client,
		accessKeyID:       accessKeyID,
		domainsCache:      f.domainsCache,
		domainsCacheMutex: &f.domainsCacheMutex,
	}, nil
}

// GetDomainNames returns a map of all domain names mapped to their composite domain names.
func (d *dnsClient) GetDomainNames(ctx context.Context) (map[string]string, error) {
	domains, err := d.getDomainsWithCache()
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
	domains, err := d.getDomainsWithCache()
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
	records, err := d.getDomainRecords(domainName, rr, recordType)
	if err != nil {
		return err
	}
	for _, value := range values {
		if record, ok := records[value]; ok {
			// Only update the existing domain record if the current TTL value is different from the given one
			// At this point we know that rr, recordType, and value are the same
			if record.TTL != ttl {
				if err := d.updateDomainRecord(record.RecordId, rr, recordType, value, ttl); err != nil {
					return err
				}
			}
		} else {
			if err := d.createDomainRecord(domainName, rr, recordType, value, ttl); err != nil {
				return err
			}
		}
	}
	for value, record := range records {
		if !utils.ValueExists(value, values) {
			if err := d.deleteDomainRecord(record.RecordId); err != nil {
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
	records, err := d.getDomainRecords(domainName, rr, recordType)
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := d.deleteDomainRecord(record.RecordId); err != nil {
			return err
		}
	}
	return nil
}

func (d *dnsClient) getDomainsWithCache() (map[string]alidns.Domain, error) {
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
	domains, err := d.getDomains()
	if err != nil {
		return nil, err
	}
	d.domainsCache.Set(d.accessKeyID, domains, domainsCacheTTL)
	return domains, nil
}

// getDomains returns all domains.
func (d *dnsClient) getDomains() (map[string]alidns.Domain, error) {
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
func (d *dnsClient) getDomainRecords(domainName, rr, recordType string) (map[string]alidns.Record, error) {
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

func (d *dnsClient) createDomainRecord(domainName, rr, recordType, value string, ttl int64) error {
	req := alidns.CreateAddDomainRecordRequest()
	req.DomainName = domainName
	req.RR = rr
	req.Type = recordType
	req.Value = value
	req.TTL = requests.NewInteger(int(ttl))
	_, err := d.Client.AddDomainRecord(req)
	return err
}

func (d *dnsClient) updateDomainRecord(id, rr, recordType, value string, ttl int64) error {
	req := alidns.CreateUpdateDomainRecordRequest()
	req.RecordId = id
	req.RR = rr
	req.Type = recordType
	req.Value = value
	req.TTL = requests.NewInteger(int(ttl))
	_, err := d.Client.UpdateDomainRecord(req)
	return err
}

func (d *dnsClient) deleteDomainRecord(id string) error {
	req := alidns.CreateDeleteDomainRecordRequest()
	req.RecordId = id
	if _, err := d.Client.DeleteDomainRecord(req); err != nil && !isDomainRecordDoesNotExistError(err) {
		return err
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
