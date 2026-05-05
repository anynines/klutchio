/*
Copyright 2026 The Klutch Bind Authors.

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

package appclusterbinding

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	appsv1 "k8s.io/api/apps/v1"
)

func TestEnqueueDeploymentOwner(t *testing.T) {
	tests := []struct {
		name        string
		deployment  *appsv1.Deployment
		expectedKey string
	}{
		{
			name: "deployment with binding labels enqueues correct key",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "konnector-my-binding",
					Namespace: "klutch-bind-my-ns-my-binding",
					Labels: map[string]string{
						appClusterBindingNameLabel:      "my-binding",
						appClusterBindingNamespaceLabel: "my-ns",
					},
				},
			},
			expectedKey: "my-ns/my-binding",
		},
		{
			name: "deployment without labels does not enqueue",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unrelated-deployment",
					Namespace: "some-namespace",
					Labels:    map[string]string{},
				},
			},
			expectedKey: "",
		},
		{
			name: "deployment with only name label does not enqueue",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "konnector-partial",
					Namespace: "klutch-bind-ns-partial",
					Labels: map[string]string{
						appClusterBindingNameLabel: "partial",
					},
				},
			},
			expectedKey: "",
		},
		{
			name: "deployment with only namespace label does not enqueue",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "konnector-partial",
					Namespace: "klutch-bind-ns-partial",
					Labels: map[string]string{
						appClusterBindingNamespaceLabel: "my-ns",
					},
				},
			},
			expectedKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			defer queue.ShutDown()

			c := &Controller{
				queue: queue,
			}

			logger := klog.Background()
			c.enqueueDeploymentOwner(logger, tt.deployment)

			if tt.expectedKey == "" {
				if queue.Len() != 0 {
					t.Errorf("expected no items in queue, got %d", queue.Len())
				}
			} else {
				if queue.Len() != 1 {
					t.Fatalf("expected 1 item in queue, got %d", queue.Len())
				}
				item, _ := queue.Get()
				key := item.(string)
				if key != tt.expectedKey {
					t.Errorf("expected key %q, got %q", tt.expectedKey, key)
				}
			}
		})
	}
}
