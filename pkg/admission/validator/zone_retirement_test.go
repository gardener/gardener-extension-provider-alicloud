// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gardener/gardener/pkg/apis/core"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/utils/ptr"

	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	apiv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
)

func TestZoneRetirementSerialization(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := api.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := apiv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	decoder := serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
	for _, tc := range []struct {
		name, field string
		disabled    bool
	}{
		{name: "old config"},
		{name: "explicit false", field: ",\"disabled\":false"},
		{name: "retirement request", field: ",\"disabled\":true", disabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := &runtime.RawExtension{Raw: []byte("{\"apiVersion\":\"alicloud.provider.extensions.gardener.cloud/v1alpha1\",\"kind\":\"InfrastructureConfig\",\"networks\":{\"vpc\":{\"cidr\":\"10.0.0.0/16\"},\"zones\":[{\"name\":\"zone1\",\"workers\":\"10.0.1.0/24\"" + tc.field + "}]}}")}
			config, err := decodeInfrastructureConfig(decoder, raw, infraConfigFldPath)
			if err != nil {
				t.Fatal(err)
			}
			if config.Networks.Zones[0].Disabled != tc.disabled {
				t.Fatalf("decoded disabled = %v", config.Networks.Zones[0].Disabled)
			}
			copied := config.DeepCopy()
			if copied.Networks.Zones[0].Disabled != tc.disabled {
				t.Fatal("deepcopy lost disabled")
			}
			external := &apiv1alpha1.InfrastructureConfig{}
			if err := scheme.Convert(copied, external, nil); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(external)
			if err != nil {
				t.Fatal(err)
			}
			var jsonConfig struct {
				Networks struct{ Zones []map[string]json.RawMessage }
			}
			if err := json.Unmarshal(encoded, &jsonConfig); err != nil {
				t.Fatal(err)
			}
			value, exists := jsonConfig.Networks.Zones[0]["disabled"]
			if exists != tc.disabled {
				t.Fatalf("disabled presence = %v, JSON: %s", exists, encoded)
			}
			if exists && string(value) != "true" {
				t.Fatalf("disabled = %s", value)
			}
			// Direct Zone conversion must also preserve the flag, independently of slice conversion.
			internalZone := api.Zone{}
			if err := scheme.Convert(&external.Networks.Zones[0], &internalZone, nil); err != nil {
				t.Fatal(err)
			}
			if internalZone.Disabled != tc.disabled {
				t.Fatal("Zone conversion lost disabled")
			}
		})
	}
}

func TestShootRejectsDisabledWorkerWithUnchangedInfrastructure(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := api.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := apiv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	s := &shoot{
		decoder:        serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder(),
		lenientDecoder: serializer.NewCodecFactory(scheme).UniversalDecoder(),
	}
	raw := []byte(`{"apiVersion":"alicloud.provider.extensions.gardener.cloud/v1alpha1","kind":"InfrastructureConfig","networks":{"vpc":{"cidr":"10.0.0.0/16"},"zones":[{"name":"zone1","workers":"10.0.1.0/24"},{"name":"zone2","workers":"10.0.2.0/24","disabled":true}]}}`)
	old := &core.Shoot{Spec: core.ShootSpec{
		Region:     "cn-hangzhou",
		Networking: &core.Networking{Nodes: ptr.To("10.0.0.0/16"), Pods: ptr.To("192.168.0.0/16"), Services: ptr.To("172.16.0.0/16")},
		Provider: core.Provider{InfrastructureConfig: &runtime.RawExtension{Raw: raw}, Workers: []core.Worker{
			{Name: "pool", Zones: []string{"zone1"}, Volume: &core.Volume{Type: ptr.To("cloud_essd"), VolumeSize: "40Gi"}},
		}},
	}}
	for _, tc := range []struct {
		zone    string
		wantErr bool
	}{{"zone1", false}, {"zone2", true}} {
		t.Run(tc.zone, func(t *testing.T) {
			obj := old.DeepCopy()
			obj.Spec.Provider.Workers = append(obj.Spec.Provider.Workers, core.Worker{
				Name: "new-pool", Zones: []string{tc.zone}, Volume: &core.Volume{Type: ptr.To("cloud_essd"), VolumeSize: "40Gi"},
			})
			// Infrastructure config is identical, so its update validator is skipped.
			err := s.validateShootUpdate(context.Background(), old, obj)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validation error: %v", err)
			}
			if tc.wantErr && !strings.Contains(err.Error(), "zone is disabled") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRetirementDoesNotLookUpFrozenBYOVSwitch(t *testing.T) {
	old := &api.InfrastructureConfig{Networks: api.Networks{Zones: []api.Zone{{Name: "zone1", WorkersVSwitchID: ptr.To("vsw-old")}}}}
	for _, alreadyDisabled := range []bool{false, true} {
		old.Networks.Zones[0].Disabled = alreadyDisabled
		config := old.DeepCopy()
		config.Networks.Zones[0].Disabled = true
		config.Networks.Zones[0].WorkersVSwitchID = ptr.To("vsw-new")
		// No cloud client or credential reader: frozen changes must reach static validation without I/O.
		gone, err := (&shoot{}).computeGoneVSwitches(context.Background(), &core.Shoot{}, old, config)
		if err != nil || len(gone) != 0 {
			t.Fatalf("gone=%v, err=%v", gone, err)
		}
	}
}

func TestShootCreationRejectsDisabledZones(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := api.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := apiv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	s := &shoot{decoder: serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()}
	for _, mode := range []string{"managed NGW", "user NGW", "BYO"} {
		for _, tc := range []struct {
			name          string
			disabledIndex int
			explicitFalse bool
		}{
			{name: "omitted", disabledIndex: -1},
			{name: "explicit false", disabledIndex: -1, explicitFalse: true},
			{name: "first zone disabled", disabledIndex: 0},
			{name: "second zone disabled", disabledIndex: 1},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				vpc := map[string]any{"cidr": "10.0.0.0/16"}
				zones := []map[string]any{{"name": "zone1", "workers": "10.0.1.0/24"}, {"name": "zone2", "workers": "10.0.2.0/24"}}
				if mode != "managed NGW" {
					vpc = map[string]any{"id": "vpc-user"}
				}
				if mode == "BYO" {
					for i, zone := range zones {
						delete(zone, "workers")
						zone["workersVSwitchID"] = "vsw-" + zones[i]["name"].(string)
					}
				}
				if tc.explicitFalse {
					for _, zone := range zones {
						zone["disabled"] = false
					}
				}
				if tc.disabledIndex >= 0 {
					zones[tc.disabledIndex]["disabled"] = true
				}
				raw, err := json.Marshal(map[string]any{
					"apiVersion": apiv1alpha1.SchemeGroupVersion.String(), "kind": "InfrastructureConfig",
					"networks": map[string]any{"vpc": vpc, "zones": zones},
				})
				if err != nil {
					t.Fatal(err)
				}
				obj := &core.Shoot{Spec: core.ShootSpec{
					Region:     "cn-hangzhou",
					Networking: &core.Networking{Nodes: ptr.To("10.0.0.0/16"), Pods: ptr.To("192.168.0.0/16"), Services: ptr.To("172.16.0.0/16")},
					Provider: core.Provider{InfrastructureConfig: &runtime.RawExtension{Raw: raw}, Workers: []core.Worker{
						{Name: "pool", Zones: []string{"zone1"}, Volume: &core.Volume{Type: ptr.To("cloud_essd"), VolumeSize: "40Gi"}},
					}},
				}}
				err = s.validateShootCreation(context.Background(), obj)
				if tc.disabledIndex < 0 {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					path := infraConfigFldPath.Child("networks", "zones").Index(tc.disabledIndex).Child("disabled").String()
					if err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "zone cannot be disabled at shoot creation") {
						t.Fatalf("expected creation rejection at %s, got %v", path, err)
					}
				}
			})
		}
	}
}
