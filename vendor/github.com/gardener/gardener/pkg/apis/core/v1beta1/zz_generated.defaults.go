//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright (c) SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by defaulter-gen. DO NOT EDIT.

package v1beta1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&CloudProfile{}, func(obj interface{}) { SetObjectDefaults_CloudProfile(obj.(*CloudProfile)) })
	scheme.AddTypeDefaultingFunc(&CloudProfileList{}, func(obj interface{}) { SetObjectDefaults_CloudProfileList(obj.(*CloudProfileList)) })
	scheme.AddTypeDefaultingFunc(&ControllerRegistration{}, func(obj interface{}) { SetObjectDefaults_ControllerRegistration(obj.(*ControllerRegistration)) })
	scheme.AddTypeDefaultingFunc(&ControllerRegistrationList{}, func(obj interface{}) { SetObjectDefaults_ControllerRegistrationList(obj.(*ControllerRegistrationList)) })
	scheme.AddTypeDefaultingFunc(&Project{}, func(obj interface{}) { SetObjectDefaults_Project(obj.(*Project)) })
	scheme.AddTypeDefaultingFunc(&ProjectList{}, func(obj interface{}) { SetObjectDefaults_ProjectList(obj.(*ProjectList)) })
	scheme.AddTypeDefaultingFunc(&SecretBinding{}, func(obj interface{}) { SetObjectDefaults_SecretBinding(obj.(*SecretBinding)) })
	scheme.AddTypeDefaultingFunc(&SecretBindingList{}, func(obj interface{}) { SetObjectDefaults_SecretBindingList(obj.(*SecretBindingList)) })
	scheme.AddTypeDefaultingFunc(&Seed{}, func(obj interface{}) { SetObjectDefaults_Seed(obj.(*Seed)) })
	scheme.AddTypeDefaultingFunc(&SeedList{}, func(obj interface{}) { SetObjectDefaults_SeedList(obj.(*SeedList)) })
	scheme.AddTypeDefaultingFunc(&Shoot{}, func(obj interface{}) { SetObjectDefaults_Shoot(obj.(*Shoot)) })
	scheme.AddTypeDefaultingFunc(&ShootList{}, func(obj interface{}) { SetObjectDefaults_ShootList(obj.(*ShootList)) })
	return nil
}

func SetObjectDefaults_CloudProfile(in *CloudProfile) {
	for i := range in.Spec.MachineImages {
		a := &in.Spec.MachineImages[i]
		for j := range a.Versions {
			b := &a.Versions[j]
			SetDefaults_MachineImageVersion(b)
		}
	}
	for i := range in.Spec.MachineTypes {
		a := &in.Spec.MachineTypes[i]
		SetDefaults_MachineType(a)
	}
	for i := range in.Spec.VolumeTypes {
		a := &in.Spec.VolumeTypes[i]
		SetDefaults_VolumeType(a)
	}
}

func SetObjectDefaults_CloudProfileList(in *CloudProfileList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_CloudProfile(a)
	}
}

func SetObjectDefaults_ControllerRegistration(in *ControllerRegistration) {
	for i := range in.Spec.Resources {
		a := &in.Spec.Resources[i]
		SetDefaults_ControllerResource(a)
		if a.Lifecycle != nil {
			SetDefaults_ControllerResourceLifecycle(a.Lifecycle)
		}
	}
	if in.Spec.Deployment != nil {
		SetDefaults_ControllerRegistrationDeployment(in.Spec.Deployment)
	}
}

func SetObjectDefaults_ControllerRegistrationList(in *ControllerRegistrationList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_ControllerRegistration(a)
	}
}

func SetObjectDefaults_Project(in *Project) {
	SetDefaults_Project(in)
	for i := range in.Spec.Members {
		a := &in.Spec.Members[i]
		SetDefaults_ProjectMember(a)
	}
}

func SetObjectDefaults_ProjectList(in *ProjectList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_Project(a)
	}
}

func SetObjectDefaults_SecretBinding(in *SecretBinding) {
	SetDefaults_SecretBinding(in)
}

func SetObjectDefaults_SecretBindingList(in *SecretBindingList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_SecretBinding(a)
	}
}

func SetObjectDefaults_Seed(in *Seed) {
	SetDefaults_Seed(in)
	SetDefaults_SeedNetworks(&in.Spec.Networks)
	if in.Spec.Settings != nil {
		if in.Spec.Settings.DependencyWatchdog != nil {
			SetDefaults_SeedSettingDependencyWatchdog(in.Spec.Settings.DependencyWatchdog)
		}
	}
}

func SetObjectDefaults_SeedList(in *SeedList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_Seed(a)
	}
}

func SetObjectDefaults_Shoot(in *Shoot) {
	SetDefaults_Shoot(in)
	if in.Spec.Addons != nil {
		if in.Spec.Addons.NginxIngress != nil {
			SetDefaults_NginxIngress(in.Spec.Addons.NginxIngress)
		}
	}
	if in.Spec.Kubernetes.ClusterAutoscaler != nil {
		SetDefaults_ClusterAutoscaler(in.Spec.Kubernetes.ClusterAutoscaler)
	}
	if in.Spec.Kubernetes.VerticalPodAutoscaler != nil {
		SetDefaults_VerticalPodAutoscaler(in.Spec.Kubernetes.VerticalPodAutoscaler)
	}
	SetDefaults_Networking(&in.Spec.Networking)
	if in.Spec.Maintenance != nil {
		SetDefaults_Maintenance(in.Spec.Maintenance)
	}
	for i := range in.Spec.Provider.Workers {
		a := &in.Spec.Provider.Workers[i]
		SetDefaults_Worker(a)
	}
}

func SetObjectDefaults_ShootList(in *ShootList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_Shoot(a)
	}
}
