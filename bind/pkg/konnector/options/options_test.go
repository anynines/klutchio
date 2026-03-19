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

package options

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		options     *CompletedOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "default-mode-valid",
			options: &CompletedOptions{
				completedOptions: &completedOptions{
					ExtraOptions: ExtraOptions{
						ControlPlaneMode: false,
					},
				},
			},
			expectError: false,
		},
		{
			name: "control-plane-mode-all-fields-valid",
			options: &CompletedOptions{
				completedOptions: &completedOptions{
					ExtraOptions: ExtraOptions{
						ControlPlaneMode:                    true,
						AppClusterKubeconfigSecretName:      "my-app-cluster",
						AppClusterKubeconfigSecretNamespace: "klutch-system",
						AppClusterKubeconfigSecretKey:       "kubeconfig",
					},
				},
			},
			expectError: false,
		},
		{
			name: "control-plane-mode-missing-name",
			options: &CompletedOptions{
				completedOptions: &completedOptions{
					ExtraOptions: ExtraOptions{
						ControlPlaneMode:                    true,
						AppClusterKubeconfigSecretNamespace: "klutch-system",
						AppClusterKubeconfigSecretKey:       "kubeconfig",
					},
				},
			},
			expectError: true,
			errorMsg:    "--app-cluster-kubeconfig-secret-name is required when --control-plane-mode is enabled",
		},
		{
			name: "control-plane-mode-missing-namespace",
			options: &CompletedOptions{
				completedOptions: &completedOptions{
					ExtraOptions: ExtraOptions{
						ControlPlaneMode:               true,
						AppClusterKubeconfigSecretName: "my-app-cluster",
						AppClusterKubeconfigSecretKey:  "kubeconfig",
					},
				},
			},
			expectError: true,
			errorMsg:    "--app-cluster-kubeconfig-secret-namespace is required when --control-plane-mode is enabled",
		},
		{
			name: "control-plane-mode-empty-key",
			options: &CompletedOptions{
				completedOptions: &completedOptions{
					ExtraOptions: ExtraOptions{
						ControlPlaneMode:                    true,
						AppClusterKubeconfigSecretName:      "my-app-cluster",
						AppClusterKubeconfigSecretNamespace: "klutch-system",
						AppClusterKubeconfigSecretKey:       "",
					},
				},
			},
			expectError: true,
			errorMsg:    "--app-cluster-kubeconfig-secret-key cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()
			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestComplete(t *testing.T) {
	tests := []struct {
		name     string
		options  *Options
		validate func(t *testing.T, completed *CompletedOptions)
	}{
		{
			name: "sets-lease-lock-identity-when-empty",
			options: &Options{
				ExtraOptions: ExtraOptions{
					LeaseLockIdentity: "",
				},
			},
			validate: func(t *testing.T, completed *CompletedOptions) {
				require.NotEmpty(t, completed.LeaseLockIdentity)
			},
		},
		{
			name: "preserves-lease-lock-identity-when-set",
			options: &Options{
				ExtraOptions: ExtraOptions{
					LeaseLockIdentity: "my-identity",
				},
			},
			validate: func(t *testing.T, completed *CompletedOptions) {
				require.Equal(t, "my-identity", completed.LeaseLockIdentity)
			},
		},
		{
			name: "preserves-control-plane-mode-settings",
			options: &Options{
				ExtraOptions: ExtraOptions{
					ControlPlaneMode:                    true,
					AppClusterKubeconfigSecretName:      "test-secret",
					AppClusterKubeconfigSecretNamespace: "test-ns",
					AppClusterKubeconfigSecretKey:       "test-key",
				},
			},
			validate: func(t *testing.T, completed *CompletedOptions) {
				require.True(t, completed.ControlPlaneMode)
				require.Equal(t, "test-secret", completed.AppClusterKubeconfigSecretName)
				require.Equal(t, "test-ns", completed.AppClusterKubeconfigSecretNamespace)
				require.Equal(t, "test-key", completed.AppClusterKubeconfigSecretKey)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completed, err := tt.options.Complete()
			require.NoError(t, err)
			tt.validate(t, completed)
		})
	}
}
