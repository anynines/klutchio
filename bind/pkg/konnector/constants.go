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

package konnector

import "strings"

const (
	// DeploymentName is the default name for the konnector deployment.
	DeploymentName = "konnector"

	// Namespace is the default namespace for the konnector.
	Namespace = "klutch-bind"

	// ContainerName is the name of the konnector container.
	ContainerName = "konnector"

	// ServiceAccountName is the default service account for the konnector.
	ServiceAccountName = "konnector"

	// AppLabelValue is the value used for the "app" label.
	AppLabelValue = "konnector"

	// HealthzPath is the readiness probe path.
	HealthzPath = "/healthz"

	// HealthzPort is the readiness probe port.
	HealthzPort = 8080

	// DefaultReplicas is the default number of replicas.
	DefaultReplicas int32 = 2

	// ImageRepository is the base image repository for the konnector.
	ImageRepository = "public.ecr.aws/w5n9a2g2/klutch/konnector"

	// Version is the default version tag for the konnector image.
	Version = "v1.4.0"
)

// Image returns the full konnector image reference for a given version tag.
func Image(version string) string {
	return ImageRepository + ":" + version
}

// ParseVersion extracts the version tag from a konnector image reference.
// Returns the version and true if the image matches ImageRepository, or
// "unknown" and false otherwise.
func ParseVersion(image string) (string, bool) {
	prefix := ImageRepository + ":"
	if !strings.HasPrefix(image, prefix) {
		return "unknown", false
	}
	return strings.TrimPrefix(image, prefix), true
}
