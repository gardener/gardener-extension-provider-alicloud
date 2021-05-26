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

package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client/ros"
	gcorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/retry"

	_ "embed"
)

// CopyImageROSTemplate contains the content of CopyImage ROS template. https://www.alibabacloud.com/help/doc-detail/116189.htm?spm=a2c63.l28256.b99.201.713413a3FkLSIx
//go:embed copyimage_ros.yaml
var CopyImageROSTemplate string

// UseEncryptedSystemDisk checks whether the given volume needs encryption
// If volume is unexpected, it will returns error to avoid unexpected result
func UseEncryptedSystemDisk(volume interface{}) (bool, error) {
	if volume == nil {
		return false, nil
	}

	if v, ok := volume.(*extensionsv1alpha1.Volume); ok {
		return v.Encrypted != nil && *v.Encrypted, nil
	}

	if v, ok := volume.(*gcorev1beta1.Volume); ok {
		return v.Encrypted != nil && *v.Encrypted, nil
	}

	return false, fmt.Errorf("type of input volume [%v] is unexpected", volume)
}

// ImageEncrypter declares interfaces to operate an encrypted image
type ImageEncrypter interface {
	TryToGetEncryptedImageID(ctx context.Context, timeout time.Duration, interval time.Duration) (string, error)
}

type imageEncryptor struct {
	regionID      string
	sourceImageID string
	imageName     string
	imageVersion  string
	rosClient     alicloudclient.ROS
}

// NewImageEncryptor creates an ImageEncrypter instance
func NewImageEncryptor(client alicloudclient.ROS, regionID, imageName, imageVersion, sourceImageID string) ImageEncrypter {
	return &imageEncryptor{
		regionID:      regionID,
		sourceImageID: sourceImageID,
		imageName:     imageName,
		imageVersion:  imageVersion,
		rosClient:     client,
	}
}

// TryToGetEncryptedImageID will get image id from stack.
// If the stack doesn't exist, it will create one and wait until the stack creation is complete
// It always takes around 10 minutes to copy an encrypted image.
// @Param timeout is the maximum time for it to wait for stack creation to complete
// @Param interval is the time period to check whether the stack is ready via REST API
func (ie *imageEncryptor) TryToGetEncryptedImageID(ctx context.Context, timeout time.Duration, interval time.Duration) (string, error) {
	stackID, err := ie.getStackIDFromName()
	if err != nil {
		return "", err
	}

	// Not found
	if stackID == "" {
		if stackID, err = ie.createStack(); err != nil {
			return "", err
		}
	}

	return ie.tryToGetEncrytpedImageIDFromStack(ctx, stackID, timeout, interval)
}

// returns imageID and error. If imageID is empty, it means the image doesn't exist
func (ie *imageEncryptor) getStackIDFromName() (string, error) {
	stackName := ie.getStackName()
	request := ros.CreateListStacksRequest()
	request.StackName = &[]string{stackName}
	request.RegionId = ie.regionID
	request.SetScheme("HTTPS")

	response, err := ie.rosClient.ListStacks(request)
	if err != nil {
		return "", err
	}

	if len(response.Stacks) == 0 {
		return "", nil
	}

	if len(response.Stacks) > 1 {
		// AliCloud doesn't allow 2 stacks with the same name. We are cautious to do a double check
		return "", fmt.Errorf("find more than 1 stacks with the same name[%s]", stackName)
	}

	return response.Stacks[0].StackId, nil
}

func (ie *imageEncryptor) createStack() (string, error) {
	stackName := ie.getStackName()

	stackRequest := ros.CreateCreateStackRequest()
	stackRequest.StackName = stackName
	stackRequest.TemplateBody = CopyImageROSTemplate
	stackRequest.Tags = &[]ros.Tag{
		{
			Key:   "gardener-managed",
			Value: "true",
		},
		{
			Key:   "do-not-delete",
			Value: "true",
		},
		{
			Key:   ie.imageName,
			Value: ie.imageVersion,
		},
	}

	parameters := []ros.CreateStackParameters{
		{ParameterKey: "ImageId", ParameterValue: ie.sourceImageID},
		{ParameterKey: "DestinationDescription", ParameterValue: fmt.Sprintf("copied from image %s", ie.sourceImageID)},
		{ParameterKey: "DestinationImageName", ParameterValue: fmt.Sprintf("%s-%s-encrypted", ie.imageName, ie.imageVersion)},
		{ParameterKey: "DestinationRegionId", ParameterValue: ie.regionID},
	}
	stackRequest.Parameters = &parameters
	stackRequest.SetScheme("HTTPS")
	response, err := ie.rosClient.CreateStack(stackRequest)
	if err != nil {
		return "", err
	}

	return response.StackId, nil
}

// This is a blocking method. It will wait around 10 minutes
func (ie *imageEncryptor) tryToGetEncrytpedImageIDFromStack(ctx context.Context, stackId string, timeout time.Duration, interval time.Duration) (string, error) {
	var imageId string
	var err error
	var needRetry bool
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = retry.Until(timeoutCtx, interval, func(ctx context.Context) (done bool, err error) {
		if imageId, needRetry, err = ie.getEncrytpedImageIDFromStack(stackId); err != nil {
			if needRetry {
				return retry.MinorError(err)
			}
			return retry.SevereError(err)
		}

		return retry.Ok()
	})

	return imageId, err
}

// This method is used for retry usage.
// It returns image id, StopRetry and error. If needRetry is false, we should do a retry if error happens.
func (ie *imageEncryptor) getEncrytpedImageIDFromStack(stackId string) (string, bool, error) {
	getStackRequest := ros.CreateGetStackRequest()
	getStackRequest.StackId = stackId
	getStackRequest.SetScheme("HTTPS")

	response, err := ie.rosClient.GetStack(getStackRequest)
	if err != nil {
		if serverErr, ok := err.(*errors.ServerError); ok {
			// Always, if it is not server error, we should not retry
			if serverErr.HttpStatus() < 500 {
				return "", false, err
			}
		}
		// Unexpected error, we should just have a retry
		return "", true, err
	}

	failedStatusSet := map[string]interface{}{
		"CREATE_FAILED":                 nil,
		"UPDATE_FAILED":                 nil,
		"DELETE_FAILED":                 nil,
		"CREATE_ROLLBACK_FAILED":        nil,
		"ROLLBACK_FAILED":               nil,
		"CHECK_FAILED":                  nil,
		"IMPORT_CREATE_FAILED":          nil,
		"IMPORT_CREATE_ROLLBACK_FAILED": nil,
		"IMPORT_UPDATE_FAILED":          nil,
		"IMPORT_UPDATE_ROLLBACK_FAILED": nil,

		"UPDATE_COMPLETE":                 nil,
		"DELETE_COMPLETE":                 nil,
		"CREATE_ROLLBACK_COMPLETE":        nil,
		"ROLLBACK_COMPLETE":               nil,
		"IMPORT_CREATE_COMPLETE":          nil,
		"IMPORT_CREATE_ROLLBACK_COMPLETE": nil,
		"IMPORT_UPDATE_COMPLETE":          nil,
		"IMPORT_UPDATE_ROLLBACK_COMPLETE": nil,
	}
	if _, ok := failedStatusSet[response.Status]; ok {
		return "", false, fmt.Errorf("the stack %s is in unexpected state: %s", stackId, response.Status)
	}

	if response.Status != "CREATE_COMPLETE" {
		return "", true, fmt.Errorf("the stack %s is still in processing state: %s", stackId, response.Status)
	}

	if len(response.Outputs) != 1 {
		return "", false, fmt.Errorf("The length output for stack %s should be 1 but got %v.\n Output is %v\n The length", stackId, len(response.Outputs), response.Outputs)
	}

	if "ImageId" != response.Outputs[0]["OutputKey"] {
		return "", false, fmt.Errorf("output doesn't contain key 'OutputKey' in stack %s", stackId)
	}

	return response.Outputs[0]["OutputValue"], false, nil
}

func (ie *imageEncryptor) getStackName() string {
	return GetEncryptImageStackName(ie.imageName, ie.imageVersion)
}

func GetEncryptImageStackName(imageName, imageVersion string) string {
	var rosNameFormat = "encrypt_image_%s_%s"
	return strings.ReplaceAll(fmt.Sprintf(rosNameFormat, imageName, imageVersion), ".", "-")
}
