// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"testing"

	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const machineTestNamespace = "shoot--test--sample"

func TestMachineUsageObserver(t *testing.T) {
	ctx := context.Background()
	targetClass := testMachineClass("class-target", "{\"vSwitchID\":\"vsw-target\"}")
	otherClass := testMachineClass("class-other", "{\"vSwitchID\":\"vsw-other\"}")

	t.Run("allows a VSwitch without residual machines or desired replicas", func(t *testing.T) {
		reader := newMachineTestReader(t, targetClass, otherClass,
			&machinev1alpha1.Machine{
				ObjectMeta: metav1.ObjectMeta{Name: "machine-other", Namespace: machineTestNamespace},
				Spec:       machinev1alpha1.MachineSpec{Class: testMachineClassRef("class-other")},
			},
			&machinev1alpha1.MachineSet{
				ObjectMeta: metav1.ObjectMeta{Name: "set-scaled-down", Namespace: machineTestNamespace},
				Spec: machinev1alpha1.MachineSetSpec{
					Replicas: 0,
					Template: machinev1alpha1.MachineTemplateSpec{Spec: machinev1alpha1.MachineSpec{Class: testMachineClassRef("class-target")}},
				},
			},
		)
		if err := newMachineUsageObserver(reader).EnsureVSwitchUnused(ctx, machineTestNamespace, "vsw-target"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("blocks a deleting Machine", func(t *testing.T) {
		deletionTimestamp := metav1.Now()
		reader := newMachineTestReader(t, targetClass, &machinev1alpha1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name: "machine-deleting", Namespace: machineTestNamespace, DeletionTimestamp: &deletionTimestamp, Finalizers: []string{"test"},
			},
			Spec: machinev1alpha1.MachineSpec{Class: testMachineClassRef("class-target")},
		})
		err := newMachineUsageObserver(reader).EnsureVSwitchUnused(ctx, machineTestNamespace, "vsw-target")
		if err == nil || !strings.Contains(err.Error(), "Machine "+machineTestNamespace+"/machine-deleting still references VSwitch vsw-target") {
			t.Fatalf("expected deleting Machine to block, got %v", err)
		}
	})

	for _, tc := range []struct {
		name   string
		object client.Object
		want   string
	}{
		{
			name: "MachineSet with desired replicas",
			object: &machinev1alpha1.MachineSet{
				ObjectMeta: metav1.ObjectMeta{Name: "set-target", Namespace: machineTestNamespace},
				Spec: machinev1alpha1.MachineSetSpec{
					Replicas: 2,
					Template: machinev1alpha1.MachineTemplateSpec{Spec: machinev1alpha1.MachineSpec{Class: testMachineClassRef("class-target")}},
				},
			},
			want: "MachineSet " + machineTestNamespace + "/set-target with 2 desired replicas still references VSwitch vsw-target",
		},
		{
			name: "MachineDeployment with desired replicas",
			object: &machinev1alpha1.MachineDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "deployment-target", Namespace: machineTestNamespace},
				Spec: machinev1alpha1.MachineDeploymentSpec{
					Replicas: 3,
					Template: machinev1alpha1.MachineTemplateSpec{Spec: machinev1alpha1.MachineSpec{Class: testMachineClassRef("class-target")}},
				},
			},
			want: "MachineDeployment " + machineTestNamespace + "/deployment-target with 3 desired replicas still references VSwitch vsw-target",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := newMachineTestReader(t, targetClass, tc.object)
			err := newMachineUsageObserver(reader).EnsureVSwitchUnused(ctx, machineTestNamespace, "vsw-target")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}

	t.Run("fails closed when a referenced MachineClass is missing", func(t *testing.T) {
		reader := newMachineTestReader(t, &machinev1alpha1.Machine{
			ObjectMeta: metav1.ObjectMeta{Name: "machine", Namespace: machineTestNamespace},
			Spec:       machinev1alpha1.MachineSpec{Class: testMachineClassRef("missing")},
		})
		err := newMachineUsageObserver(reader).EnsureVSwitchUnused(ctx, machineTestNamespace, "vsw-target")
		if err == nil || !strings.Contains(err.Error(), "get MachineClass "+machineTestNamespace+"/missing") {
			t.Fatalf("expected missing class error, got %v", err)
		}
	})

	t.Run("fails closed when providerSpec cannot be parsed", func(t *testing.T) {
		invalidClass := testMachineClass("class-invalid", "{\"vSwitchID\":123}")
		reader := newMachineTestReader(t, invalidClass, &machinev1alpha1.Machine{
			ObjectMeta: metav1.ObjectMeta{Name: "machine", Namespace: machineTestNamespace},
			Spec:       machinev1alpha1.MachineSpec{Class: testMachineClassRef("class-invalid")},
		})
		err := newMachineUsageObserver(reader).EnsureVSwitchUnused(ctx, machineTestNamespace, "vsw-target")
		if err == nil || !strings.Contains(err.Error(), "decode MachineClass") {
			t.Fatalf("expected decode error, got %v", err)
		}
	})

	t.Run("fails closed for an unsupported class reference", func(t *testing.T) {
		reader := newMachineTestReader(t, &machinev1alpha1.Machine{
			ObjectMeta: metav1.ObjectMeta{Name: "machine", Namespace: machineTestNamespace},
			Spec: machinev1alpha1.MachineSpec{Class: machinev1alpha1.ClassSpec{
				Name: "class-target", Kind: "OtherClass", APIGroup: "other.example.io",
			}},
		})
		err := newMachineUsageObserver(reader).EnsureVSwitchUnused(ctx, machineTestNamespace, "vsw-target")
		if err == nil || !strings.Contains(err.Error(), "unsupported kind") {
			t.Fatalf("expected class reference error, got %v", err)
		}
	})

	t.Run("propagates API list failures", func(t *testing.T) {
		baseReader := newMachineTestReader(t)
		observer := newMachineUsageObserver(&failingMachineListReader{Reader: baseReader})
		err := observer.EnsureVSwitchUnused(ctx, machineTestNamespace, "vsw-target")
		if err == nil || !strings.Contains(err.Error(), "list Machines") || !strings.Contains(err.Error(), "forbidden") {
			t.Fatalf("expected list failure, got %v", err)
		}
	})
}

type failingMachineListReader struct {
	client.Reader
}

func (r *failingMachineListReader) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	if _, ok := list.(*machinev1alpha1.MachineList); ok {
		return fmt.Errorf("forbidden")
	}
	return r.Reader.List(context.Background(), list)
}

func newMachineTestReader(t *testing.T, objects ...client.Object) client.Reader {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := machinev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func testMachineClass(name, providerSpec string) *machinev1alpha1.MachineClass {
	return &machinev1alpha1.MachineClass{
		ObjectMeta:   metav1.ObjectMeta{Name: name, Namespace: machineTestNamespace},
		ProviderSpec: runtime.RawExtension{Raw: []byte(providerSpec)},
	}
}

func testMachineClassRef(name string) machinev1alpha1.ClassSpec {
	return machinev1alpha1.ClassSpec{Name: name, Kind: machineClassKind}
}
