/*
Copyright 2022 The Kube Bind Authors.

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

package resources

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
	bindclient "github.com/anynines/klutchio/bind/pkg/client/clientset/versioned"
)

func CreateClusterBinding(ctx context.Context, client bindclient.Interface, ns, secretName, providerPrettyName string) error {
	logger := klog.FromContext(ctx)

	clusterBinding := &bindv1alpha1.ClusterBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterBindingName,
			Namespace: ns,
		},
		Spec: bindv1alpha1.ClusterBindingSpec{
			ProviderPrettyName: providerPrettyName,
			KubeconfigSecretRef: bindv1alpha1.LocalSecretKeyRef{
				Name: secretName,
				Key:  "kubeconfig",
			},
		},
	}

	logger.V(3).Info("Creating ClusterBinding")
	_, err := client.KlutchBindV1alpha1().ClusterBindings(ns).Create(ctx, clusterBinding, metav1.CreateOptions{})
	return err
}
