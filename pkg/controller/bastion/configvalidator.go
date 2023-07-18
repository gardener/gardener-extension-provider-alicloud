package bastion

import (
	"context"
	"errors"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	aliclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	alicloudapi "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
)

// configValidator implements ConfigValidator for AliCloud bastion resources.
type configValidator struct {
	client           client.Client
	aliClientFactory aliclient.ClientFactory
}

// NewConfigValidator creates a new ConfigValidator.
func NewConfigValidator(aliClientFactory aliclient.ClientFactory) bastion.ConfigValidator {
	return &configValidator{
		aliClientFactory: aliClientFactory,
	}
}

// Validate validates the provider config of the given bastion resource with the cloud provider.
func (c *configValidator) Validate(ctx context.Context, _ *extensionsv1alpha1.Bastion, cluster *extensions.Cluster) field.ErrorList {
	allErrs := field.ErrorList{}

	// Get value from infrastructure status
	infrastructureStatus, err := getInfrastructureStatus(ctx, c.client, cluster)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, err))
		return allErrs
	}

	secretReference := &corev1.SecretReference{
		Namespace: cluster.ObjectMeta.Name,
		Name:      v1beta1constants.SecretNameCloudProvider,
	}

	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, c.client, secretReference)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, err))
		return allErrs
	}

	// Create alicloud ECS client
	aliCloudECSClient, err := c.aliClientFactory.NewECSClient(cluster.Shoot.Spec.Region, credentials.AccessKeyID, credentials.AccessKeySecret)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, err))
		return allErrs
	}

	aliCloudVPCClient, err := c.aliClientFactory.NewVPCClient(cluster.Shoot.Spec.Region, credentials.AccessKeyID, credentials.AccessKeySecret)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, err))
		return allErrs
	}

	// Validate infrastructureStatus value
	allErrs = append(allErrs, c.validateInfrastructureStatus(ctx, aliCloudECSClient, aliCloudVPCClient, infrastructureStatus)...)
	return allErrs
}

func getInfrastructureStatus(ctx context.Context, c client.Client, cluster *extensions.Cluster) (*alicloudapi.InfrastructureStatus, error) {
	var infrastructureStatus *alicloudapi.InfrastructureStatus
	worker := &extensionsv1alpha1.Worker{}
	err := c.Get(ctx, client.ObjectKey{Namespace: cluster.ObjectMeta.Name, Name: cluster.Shoot.Name}, worker)
	if err != nil {
		return nil, err
	}

	if worker.Spec.InfrastructureProviderStatus == nil {
		return nil, errors.New("infrastructure provider status must be not empty for worker")
	}

	if infrastructureStatus, err = helper.InfrastructureStatusFromRaw(worker.Spec.InfrastructureProviderStatus); err != nil {
		return nil, err
	}

	if infrastructureStatus.VPC.ID == "" {
		return nil, errors.New("vpc id must be not empty for infrastructure provider status")
	}

	if len(infrastructureStatus.VPC.VSwitches) == 0 || infrastructureStatus.VPC.VSwitches[0].ID == "" || infrastructureStatus.VPC.VSwitches[0].Zone == "" {
		return nil, errors.New("vswitches id must be not empty for infrastructure provider status")
	}

	if len(infrastructureStatus.MachineImages) == 0 || infrastructureStatus.MachineImages[0].ID == "" {
		return nil, errors.New("machineImages id must be not empty for infrastructure provider status")
	}

	// The assumption is that the shoot only has one security group
	if len(infrastructureStatus.VPC.SecurityGroups) == 0 || infrastructureStatus.VPC.SecurityGroups[0].ID == "" {
		return nil, errors.New("shoot securityGroups id must be not empty for infrastructure provider status")
	}

	return infrastructureStatus, nil
}

func (c *configValidator) validateInfrastructureStatus(ctx context.Context, aliCloudECSClient aliclient.ECS, aliCloudVPCClient aliclient.VPC, infrastructureStatus *alicloudapi.InfrastructureStatus) field.ErrorList {
	allErrs := field.ErrorList{}

	vpc, err := aliCloudVPCClient.GetVPCWithID(ctx, infrastructureStatus.VPC.ID)
	if err != nil || len(vpc) == 0 {
		allErrs = append(allErrs, field.InternalError(field.NewPath("vpc"), fmt.Errorf("could not get vpc %s from alicloud provider: %w", infrastructureStatus.VPC.ID, err)))
		return allErrs
	}

	vSwitch, err := aliCloudVPCClient.GetVSwitchesInfoByID(infrastructureStatus.VPC.VSwitches[0].ID)
	if err != nil || vSwitch.ZoneID == "" {
		allErrs = append(allErrs, field.InternalError(field.NewPath("vswitches"), fmt.Errorf("could not get vswitches %s from alicloud provider: %w", infrastructureStatus.VPC.VSwitches[0].ID, err)))
		return allErrs
	}

	machineImages, err := aliCloudECSClient.CheckIfImageExists(infrastructureStatus.MachineImages[0].ID)
	if err != nil || !machineImages {
		allErrs = append(allErrs, field.InternalError(field.NewPath("machineImages"), fmt.Errorf("could not get machineImages %s from alicloud provider: %w", infrastructureStatus.MachineImages[0].ID, err)))
		return allErrs
	}

	shootSecurityGroupId, err := aliCloudECSClient.GetSecurityGroupWithID(infrastructureStatus.VPC.SecurityGroups[0].ID)
	if err != nil || len(shootSecurityGroupId.SecurityGroups.SecurityGroup) == 0 || shootSecurityGroupId.SecurityGroups.SecurityGroup[0].SecurityGroupId == "" {
		allErrs = append(allErrs, field.InternalError(field.NewPath("securityGroup"), fmt.Errorf("could not get shoot security group %s from alicloud provider: %w", infrastructureStatus.VPC.SecurityGroups[0].ID, err)))
		return allErrs
	}

	return allErrs
}
