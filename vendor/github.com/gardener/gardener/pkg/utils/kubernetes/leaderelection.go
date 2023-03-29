// Copyright 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReadLeaderElectionRecord returns the leader election record for a given lock type and a namespace/name combination.
func ReadLeaderElectionRecord(ctx context.Context, client client.Client, lock, namespace, name string) (*resourcelock.LeaderElectionRecord, error) {
	switch lock {
	case "endpoints":
		endpoint := &corev1.Endpoints{}
		if err := client.Get(ctx, Key(namespace, name), endpoint); err != nil {
			return nil, err
		}
		return leaderElectionRecordFromAnnotations(endpoint.Annotations)

	case "configmaps":
		configmap := &corev1.ConfigMap{}
		if err := client.Get(ctx, Key(namespace, name), configmap); err != nil {
			return nil, err
		}
		return leaderElectionRecordFromAnnotations(configmap.Annotations)

	case resourcelock.LeasesResourceLock:
		lease := &coordinationv1.Lease{}
		if err := client.Get(ctx, Key(namespace, name), lease); err != nil {
			return nil, err
		}
		return resourcelock.LeaseSpecToLeaderElectionRecord(&lease.Spec), nil
	}

	return nil, fmt.Errorf("unknown lock type: %s", lock)
}

func leaderElectionRecordFromAnnotations(annotations map[string]string) (*resourcelock.LeaderElectionRecord, error) {
	var leaderElectionRecord resourcelock.LeaderElectionRecord

	leaderElection, ok := annotations[resourcelock.LeaderElectionRecordAnnotationKey]
	if !ok {
		return nil, fmt.Errorf("could not find key %q in annotations", resourcelock.LeaderElectionRecordAnnotationKey)
	}

	if err := json.Unmarshal([]byte(leaderElection), &leaderElectionRecord); err != nil {
		return nil, fmt.Errorf("failed to unmarshal leader election record: %+v", err)
	}

	return &leaderElectionRecord, nil
}
