// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
)

func TestGetRR(t *testing.T) {
	tests := []struct {
		name       string
		dnsName    string
		domainName string
		expected   string
		expectErr  bool
	}{
		{
			name:       "simple subdomain",
			dnsName:    "api.shoot.example.com",
			domainName: "shoot.example.com",
			expected:   "api",
		},
		{
			name:       "nested subdomain",
			dnsName:    "orc.hc-cn40can.test.canary-cn40.hanacloud.example.cn",
			domainName: "test.canary-cn40.hanacloud.example.cn",
			expected:   "orc.hc-cn40can",
		},
		{
			name:       "same as domain returns @",
			dnsName:    "shoot.example.com",
			domainName: "shoot.example.com",
			expected:   "@",
		},
		{
			name:       "mismatched domain returns error",
			dnsName:    "api.other.com",
			domainName: "shoot.example.com",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getRR(tt.dnsName, tt.domainName)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFilterRecordsByExactRR(t *testing.T) {
	// This test validates the filtering logic that getDomainRecords applies.
	// The Alicloud DescribeDomainRecords API uses RRKeyWord for fuzzy/substring matching,
	// so responses may contain records that don't exactly match the requested RR.
	// getDomainRecords must filter for exact RR match.

	targetRR := "tnt.hc-cn40can"

	apiResponse := []alidns.Record{
		{RR: "tnt.hc-cn40can", Value: "1.2.3.4", RecordId: "1"},
		{RR: "orc.hc-cn40can.tnt.hc-cn40can", Value: "5.6.7.8", RecordId: "2"}, // fuzzy match - contains "tnt.hc-cn40can"
		{RR: "tnt.hc-cn40can", Value: "9.10.11.12", RecordId: "3"},
		{RR: "haas.tnt.hc-cn40can", Value: "13.14.15.16", RecordId: "4"}, // fuzzy match
	}

	// Simulate the filtering logic from getDomainRecords
	records := make(map[string]alidns.Record)
	for _, record := range apiResponse {
		if record.RR == targetRR {
			records[record.Value] = record
		}
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records after filtering, got %d", len(records))
	}
	if _, ok := records["1.2.3.4"]; !ok {
		t.Error("expected record with value 1.2.3.4")
	}
	if _, ok := records["9.10.11.12"]; !ok {
		t.Error("expected record with value 9.10.11.12")
	}
	if _, ok := records["5.6.7.8"]; ok {
		t.Error("record with value 5.6.7.8 should have been filtered out")
	}
	if _, ok := records["13.14.15.16"]; ok {
		t.Error("record with value 13.14.15.16 should have been filtered out")
	}
}

func TestCompositeDomainName(t *testing.T) {
	tests := []struct {
		name       string
		domainName string
		domainId   string
		expected   string
	}{
		{"with id", "example.com", "123", "example.com:123"},
		{"without id", "example.com", "", "example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompositeDomainName(tt.domainName, tt.domainId)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDomainNameAndId(t *testing.T) {
	tests := []struct {
		name           string
		composite      string
		expectedName   string
		expectedId     string
	}{
		{"with id", "example.com:123", "example.com", "123"},
		{"without id", "example.com", "example.com", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, id := DomainNameAndId(tt.composite)
			if name != tt.expectedName {
				t.Errorf("expected name %q, got %q", tt.expectedName, name)
			}
			if id != tt.expectedId {
				t.Errorf("expected id %q, got %q", tt.expectedId, id)
			}
		})
	}
}
