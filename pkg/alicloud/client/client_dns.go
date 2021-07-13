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

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/gardener/gardener/pkg/utils"
)

// NewDNSClient creates a new DNS client with given region, accessKeyID, and accessKeySecret.
func (f *clientFactory) NewDNSClient(region, accessKeyID, accessKeySecret string) (DNS, error) {
	client, err := alidns.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}

	return &dnsClient{
		*client,
	}, nil
}

// GetDomainNames returns a list of all domain names.
func (d *dnsClient) GetDomainNames(ctx context.Context) ([]string, error) {
	var domains []string
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
			domains = append(domains, domain.DomainName)
		}
		if resp.PageNumber*int64(pageSize) >= resp.TotalCount {
			break
		}
		pageNumber++
	}
	return domains, nil
}

// CreateOrUpdateDomainRecords creates or updates the domain records with the given domain name, name, record type,
// values, and ttl.
// * For each element in values that has an existing domain record, the existing record is updated if needed.
// * For each element in values that doesn't have an existing domain record, a new domain record is created.
// * For each existing domain record that doesn't have a corresponding element in values, the existing record is deleted.
func (d *dnsClient) CreateOrUpdateDomainRecords(ctx context.Context, domainName, name, recordType string, values []string, ttl int64) error {
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
