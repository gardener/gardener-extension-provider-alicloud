// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const machineClassKind = "MachineClass"

type machineUsageObserver struct {
	reader client.Reader
}

func newMachineUsageObserver(reader client.Reader) *machineUsageObserver {
	return &machineUsageObserver{reader: reader}
}

func (o *machineUsageObserver) EnsureVSwitchUnused(ctx context.Context, namespace, vSwitchID string) error {
	if o == nil || o.reader == nil {
		return fmt.Errorf("machine API reader is not configured")
	}

	classVSwitches := map[string]string{}
	checkClass := func(ref machinev1alpha1.ClassSpec, source string) error {
		classKey := ref.APIGroup + "/" + ref.Kind + "/" + ref.Name
		classVSwitchID, ok := classVSwitches[classKey]
		if !ok {
			var err error
			classVSwitchID, err = o.machineClassVSwitchID(ctx, namespace, ref)
			if err != nil {
				return fmt.Errorf("cannot verify %s: %w", source, err)
			}
			classVSwitches[classKey] = classVSwitchID
		}
		if classVSwitchID == vSwitchID {
			return fmt.Errorf("%s still references VSwitch %s", source, vSwitchID)
		}
		return nil
	}

	machines := &machinev1alpha1.MachineList{}
	if err := o.reader.List(ctx, machines, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("list Machines in namespace %s: %w", namespace, err)
	}
	for i := range machines.Items {
		machine := &machines.Items[i]
		if err := checkClass(machine.Spec.Class, fmt.Sprintf("Machine %s/%s", namespace, machine.Name)); err != nil {
			return err
		}
	}

	machineSets := &machinev1alpha1.MachineSetList{}
	if err := o.reader.List(ctx, machineSets, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("list MachineSets in namespace %s: %w", namespace, err)
	}
	for i := range machineSets.Items {
		machineSet := &machineSets.Items[i]
		if machineSet.Spec.Replicas <= 0 {
			continue
		}
		ref := machineSet.Spec.Template.Spec.Class
		if ref.Name == "" {
			ref = machineSet.Spec.MachineClass
		}
		if err := checkClass(ref, fmt.Sprintf("MachineSet %s/%s with %d desired replicas", namespace, machineSet.Name, machineSet.Spec.Replicas)); err != nil {
			return err
		}
	}

	machineDeployments := &machinev1alpha1.MachineDeploymentList{}
	if err := o.reader.List(ctx, machineDeployments, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("list MachineDeployments in namespace %s: %w", namespace, err)
	}
	for i := range machineDeployments.Items {
		machineDeployment := &machineDeployments.Items[i]
		if machineDeployment.Spec.Replicas <= 0 {
			continue
		}
		if err := checkClass(
			machineDeployment.Spec.Template.Spec.Class,
			fmt.Sprintf("MachineDeployment %s/%s with %d desired replicas", namespace, machineDeployment.Name, machineDeployment.Spec.Replicas),
		); err != nil {
			return err
		}
	}

	return nil
}

func (o *machineUsageObserver) machineClassVSwitchID(ctx context.Context, namespace string, ref machinev1alpha1.ClassSpec) (string, error) {
	if ref.Name == "" {
		return "", fmt.Errorf("machine class reference has no name")
	}
	if ref.Kind != machineClassKind {
		return "", fmt.Errorf("machine class %q has unsupported kind %q", ref.Name, ref.Kind)
	}
	if ref.APIGroup != "" && ref.APIGroup != machinev1alpha1.GroupName {
		return "", fmt.Errorf("machine class %q has unsupported API group %q", ref.Name, ref.APIGroup)
	}

	machineClass := &machinev1alpha1.MachineClass{}
	if err := o.reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ref.Name}, machineClass); err != nil {
		return "", fmt.Errorf("get MachineClass %s/%s: %w", namespace, ref.Name, err)
	}

	raw, err := rawExtensionBytes(machineClass.ProviderSpec)
	if err != nil {
		return "", fmt.Errorf("encode MachineClass %s/%s providerSpec: %w", namespace, ref.Name, err)
	}
	providerSpec := struct {
		VSwitchID string `json:"vSwitchID"`
	}{}
	if err := json.Unmarshal(raw, &providerSpec); err != nil {
		return "", fmt.Errorf("decode MachineClass %s/%s providerSpec: %w", namespace, ref.Name, err)
	}
	if providerSpec.VSwitchID == "" {
		return "", fmt.Errorf("MachineClass %s/%s providerSpec has no vSwitchID", namespace, ref.Name)
	}
	return providerSpec.VSwitchID, nil
}

func rawExtensionBytes(rawExtension runtime.RawExtension) ([]byte, error) {
	if len(rawExtension.Raw) > 0 {
		return rawExtension.Raw, nil
	}
	if rawExtension.Object == nil {
		return nil, fmt.Errorf("providerSpec is empty")
	}
	return json.Marshal(rawExtension.Object)
}
