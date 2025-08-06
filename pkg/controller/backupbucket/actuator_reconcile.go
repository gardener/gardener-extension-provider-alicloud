// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package backupbucket

import (
	"context"
	"fmt"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/admission/validator"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	apisali "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	"github.com/gardener/gardener/extensions/pkg/util"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// Reconcile reconciles the BackupBucket resource with following steps:
// 1. Create alicloud's oss client from secret ref.
// 2. Decode the backupbucket config (if provided).
// 3. Check if bucket already exist or not.
// 4. If bucket doesn't exist
//   - then create a new bucket according to backupbucketConfig, if provided.
//
// 5. If bucket exist
//   - check for bucket update is required or not
//   - If yes then update the backup bucket settings according to backupbucketConfig(if provided)
//     otherwise do nothing.
func (a *actuator) Reconcile(ctx context.Context, logger logr.Logger, bb *extensionsv1alpha1.BackupBucket) error {
	logger.Info("Starting reconciliation of BackupBucket...")

	authConfig, err := alicloud.ReadCredentialsFromSecretRef(ctx, a.client, &bb.Spec.SecretRef)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	ossClient, err := a.aliClientFactory.NewOSSClient(alicloudclient.ComputeStorageEndpoint(bb.Spec.Region), authConfig.AccessKeyID, authConfig.AccessKeySecret)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	var backupBucketConfig *apisali.BackupBucketConfig
	if bb.Spec.ProviderConfig != nil {
		backupBucketConfig, err = validator.DecodeBackupBucketConfig(serializer.NewCodecFactory(a.client.Scheme(), serializer.EnableStrict).UniversalDecoder(), bb.Spec.ProviderConfig)
		if err != nil {
			return util.DetermineError(fmt.Errorf("failed to decode provider config: %w", err), helper.KnownCodes)
		}
	}

	a.action = ActionFunc(getAction(ossClient, bb.Name, backupBucketConfig))

	return util.DetermineError(a.reconcile(ctx, logger, ossClient, bb.Name, backupBucketConfig), helper.KnownCodes)
}

func (a *actuator) reconcile(ctx context.Context, _ logr.Logger, ossClient alicloudclient.OSS, bucket string, backupBucketConfig *apisali.BackupBucketConfig) error {
	if _, err := ossClient.GetBucketInfo(bucket); err != nil {
		ossErr, ok := err.(oss.ServiceError)
		if !ok {
			return util.DetermineError(err, helper.KnownCodes)
		}
		if ossErr.Code == "NoSuchBucket" {
			if err := ossClient.CreateBucketIfNotExists(ctx, bucket); err != nil {
				return util.DetermineError(err, helper.KnownCodes)
			}
		}
	}

	if isBucketLockConfigNeedToBeRemoved(ossClient, bucket, backupBucketConfig) {
		return util.DetermineError(ossClient.AbortRetentionPolcy(bucket), helper.KnownCodes)
	}
	if isBucketUpdateRequired(ossClient, bucket, backupBucketConfig) {
		// take action: update the bucket with bucket lock settings.
		return util.DetermineError(a.action.Do(), helper.KnownCodes)
	}

	return nil
}

func isBucketUpdateRequired(ossClient alicloudclient.OSS, bucket string, backupbucketConfig *apisali.BackupBucketConfig) bool {
	if backupbucketConfig == nil || backupbucketConfig.Immutability == nil {
		return false
	}

	wormConfig, err := ossClient.GetBucketWorm(bucket)
	if err != nil {
		// if bucket lock configurations aren't set on bucket
		// then bucket update is required
		return true
	}

	if wormConfig != nil {
		switch wormConfig.State {
		case "Expired":
			if backupbucketConfig.Immutability != nil {
				return true
			}

		case "Locked":
			if wormConfig.RetentionPeriodInDays != backupbucketConfig.Immutability.RetentionPeriod {
				return true
			}

		case "InProgress":
			// Note: In Worm state: "InProgress", retentionPeriod can't be extended until retentionPolicy is locked.
			if backupbucketConfig.Immutability.Locked {
				return true
			}
		}
	}
	return false
}

func isBucketLockConfigNeedToBeRemoved(ossClient alicloudclient.OSS, bucket string, backupbucketConfig *apisali.BackupBucketConfig) bool {
	wormConfig, err := ossClient.GetBucketWorm(bucket)
	if err != nil {
		// worm configuration doesn't exist for given bucket.
		return false
	}

	if wormConfig != nil {
		switch wormConfig.State {
		case "Locked":
			// once retentionPolicy is locked, bucket lock can't be removed
			return false

		case "InProgress", "Expired":
			if backupbucketConfig == nil || backupbucketConfig.Immutability == nil {
				return true
			}
		}
	}

	return false
}

func getAction(ossClient alicloudclient.OSS, bucket string, backupBucketConfig *apisali.BackupBucketConfig) func() error {
	return func() error {
		// enable or update the bucket lock configuration on the bucket
		// by adding or updating the retentionPolicy on the bucket.
		wormConfig, err := ossClient.GetBucketWorm(bucket)
		if err != nil {
			if ossErr, ok := err.(oss.ServiceError); !ok {
				return err
			} else if ossErr.Code == "NoSuchWORMConfiguration" {
				wormID, err := ossClient.CreateRetentionPolicy(bucket, backupBucketConfig.Immutability.RetentionPeriod)
				if err != nil {
					return err
				}
				if backupBucketConfig.Immutability.Locked {
					if err := ossClient.LockRetentionPolicy(bucket, wormID); err != nil {
						return err
					}
				}
				return nil
			}
		}

		if wormConfig != nil {
			switch wormConfig.State {
			case "Expired":
				// first remove the expired retentionPolicy
				// then only a new retentionPolicy can be added.
				if err := ossClient.AbortRetentionPolcy(bucket); err != nil {
					return err
				}
				wormID, err := ossClient.CreateRetentionPolicy(bucket, backupBucketConfig.Immutability.RetentionPeriod)
				if err != nil {
					return err
				}
				if backupBucketConfig.Immutability.Locked {
					if err := ossClient.LockRetentionPolicy(bucket, wormID); err != nil {
						return err
					}
				}

			case "Locked":
				if err := ossClient.UpdateRetentionPolicy(bucket, backupBucketConfig.Immutability.RetentionPeriod, wormConfig.WormId); err != nil {
					return err
				}

			case "InProgress":
				// Note: In Worm state: "InProgress", retentionPeriod can't be extended until retentionPolicy is locked.
				if backupBucketConfig.Immutability.Locked {
					if err := ossClient.LockRetentionPolicy(bucket, wormConfig.WormId); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
}
