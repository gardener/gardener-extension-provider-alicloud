// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package bastion

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"

	"github.com/gardener/gardener/extensions/pkg/controller"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const (
	//maxLengthForBaseName for "base" name due to fact that we use this name to name other Alicloud resources,
	maxLengthForBaseName = 33
)

// var cores int = 1

type etherType int64

const (
	ipv4Type etherType = iota
	ipv6Type
)

// Options contains provider-related information required for setting up
// a bastion instance. This struct combines precomputed values like the
// bastion instance name with the IDs of pre-existing cloud provider
// resources, like the vpc name, shoot security group name etc.
type Options struct {
	BastionInstanceName string
	Region              string
	SecretReference     corev1.SecretReference
	SecurityGroupName   string
	ShootName           string
	UserData            string
}

// DetermineOptions determines the information that are required to reconcile a Bastion on Alicloud. This
// function does not create any IaaS resources.
func DetermineOptions(bastion *extensionsv1alpha1.Bastion, cluster *controller.Cluster) (*Options, error) {
	clusterName := cluster.ObjectMeta.Name
	region := cluster.Shoot.Spec.Region

	baseResourceName, err := generateBastionBaseResourceName(clusterName, bastion.Name)
	if err != nil {
		return nil, err
	}

	secretReference := corev1.SecretReference{
		Namespace: cluster.ObjectMeta.Name,
		Name:      v1beta1constants.SecretNameCloudProvider,
	}

	return &Options{
		ShootName:           clusterName,
		BastionInstanceName: baseResourceName,
		SecretReference:     secretReference,
		Region:              region,
		SecurityGroupName:   securityGroupName(baseResourceName),
		UserData:            base64.StdEncoding.EncodeToString(bastion.Spec.UserData),
	}, nil
}

func generateBastionBaseResourceName(clusterName string, bastionName string) (string, error) {
	if clusterName == "" {
		return "", fmt.Errorf("clusterName can't be empty")
	}
	if bastionName == "" {
		return "", fmt.Errorf("bastionName can't be empty")
	}

	staticName := clusterName + "-" + bastionName
	h := sha256.New()
	_, err := h.Write([]byte(staticName))
	if err != nil {
		return "", err
	}
	hash := fmt.Sprintf("%x", h.Sum(nil))
	if len([]rune(staticName)) > maxLengthForBaseName {
		staticName = staticName[:maxLengthForBaseName]
	}
	return fmt.Sprintf("%s-bastion-%s", staticName, hash[:5]), nil
}

// securityGroupName is security group name
func securityGroupName(baseName string) string {
	return fmt.Sprintf("%s-sg", baseName)
}

// IngressPermission holds the IPv4 and IPv6 ranges that should be allowed to access the bastion.
type IngressPermission struct {
	// EtherType describes is ipv4 or ipv6 ether type
	EtherType etherType

	// CIDR holds the IPv4 or IPv6 range, depending on EtherType.
	CIDR string
}

func ingressPermissions(bastion *extensionsv1alpha1.Bastion) ([]IngressPermission, error) {
	var perms []IngressPermission

	for _, ingress := range bastion.Spec.Ingress {
		cidr := ingress.IPBlock.CIDR
		ip, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid ingress CIDR %q: %w", cidr, err)
		}

		normalisedCIDR := ipNet.String()

		if ip.To4() != nil {
			perms = append(perms, IngressPermission{EtherType: ipv4Type, CIDR: normalisedCIDR})
		} else if ip.To16() != nil {
			perms = append(perms, IngressPermission{EtherType: ipv6Type, CIDR: normalisedCIDR})
		}
	}
	return perms, nil
}
