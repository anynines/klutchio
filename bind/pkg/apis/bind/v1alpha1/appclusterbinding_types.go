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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	conditionsapi "github.com/anynines/klutchio/bind/pkg/apis/third_party/conditions/apis/conditions/v1alpha1"
)

const (
	// AppClusterBindingConditionSecretValid is set when the kubeconfig secret is valid.
	AppClusterBindingConditionSecretValid conditionsapi.ConditionType = "SecretValid"
	// AppClusterBindingConditionKonnectorDeployed is set when the konnector deployment is created/updated.
	AppClusterBindingConditionKonnectorDeployed conditionsapi.ConditionType = "KonnectorDeployed"
)

// AppClusterBinding represents a binding for an app cluster on the control plane.
//
// +crd
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Namespaced,categories=kube-bindings
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.conditions[?(@.type=="Ready")].status`,priority=0
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=`.metadata.creationTimestamp`,priority=0
// +kubebuilder:validation:XValidation:rule="self.metadata.name == oldSelf.metadata.name",message="name is immutable"
type AppClusterBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec represents the desired state of the AppClusterBinding.
	// +required
	// +kubebuilder:validation:Required
	Spec AppClusterBindingSpec `json:"spec"`

	// status contains reconciliation information.
	Status AppClusterBindingStatus `json:"status,omitempty"`
}

func (in *AppClusterBinding) GetConditions() conditionsapi.Conditions {
	return in.Status.Conditions
}

func (in *AppClusterBinding) SetConditions(conditions conditionsapi.Conditions) {
	in.Status.Conditions = conditions
}

// AppClusterBindingSpec represents the desired state of an AppClusterBinding.
type AppClusterBindingSpec struct {
	// kubeconfigSecretRef points to the app cluster kubeconfig.
	//
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="kubeconfigSecretRef is immutable"
	KubeconfigSecretRef ClusterSecretKeyRef `json:"kubeconfigSecretRef"`

	// apiExports is a list of GroupResource entries, where each entry specifies an API group and resource to bind.
	//
	// +optional
	APIExports []GroupResource `json:"apiExports,omitempty"`

	// konnector contains deployment settings for the konnector.
	//
	// +optional
	Konnector *KonnectorSpec `json:"konnector,omitempty"`
}

// KonnectorSpec controls konnector deployment behavior.
type KonnectorSpec struct {
	// deploy enables konnector deployment for this binding.
	//
	// +optional
	Deploy bool `json:"deploy,omitempty"`

	// overrides allow modifying the container spec for the konnector.
	// Fields set here are strategically merged into the base konnector container.
	//
	// +optional
	Overrides *ContainerOverrides `json:"overrides,omitempty"`
}

// ContainerOverrides defines optional container-level overrides for the konnector deployment.
// These are applied to the konnector container via strategic merge patch.
// All fields are optional and mirror corev1.Container.
type ContainerOverrides struct {
	// Name of the container specified as a DNS_LABEL.
	// Each container in a pod must have a unique name (DNS_LABEL).
	// +optional
	Name string `json:"name,omitempty"`
	// Container image name.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	// +optional
	Image string `json:"image,omitempty"`
	// Entrypoint array. Not executed within a shell.
	// The container image's ENTRYPOINT is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	// +listType=atomic
	Command []string `json:"command,omitempty"`
	// Arguments to the entrypoint.
	// The container image's CMD is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	// +listType=atomic
	Args []string `json:"args,omitempty"`
	// Container's working directory.
	// If not specified, the container runtime's default will be used, which
	// might be configured in the container image.
	// +optional
	WorkingDir string `json:"workingDir,omitempty"`
	// List of ports to expose from the container. Not specifying a port here
	// DOES NOT prevent that port from being exposed.
	// +optional
	// +patchMergeKey=containerPort
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=containerPort
	// +listMapKey=protocol
	Ports []corev1.ContainerPort `json:"ports,omitempty" patchStrategy:"merge" patchMergeKey:"containerPort"`
	// List of sources to populate environment variables in the container.
	// The keys defined within a source must be a C_IDENTIFIER.
	// +optional
	// +listType=atomic
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`
	// List of environment variables to set in the container.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	Env []corev1.EnvVar `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	// Compute Resources required by this container.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// RestartPolicy defines the restart behavior of individual containers in a pod.
	// This is only effective for init containers with the sidecar container feature.
	// +optional
	RestartPolicy *corev1.ContainerRestartPolicy `json:"restartPolicy,omitempty"`
	// Pod volumes to mount into the container's filesystem.
	// +optional
	// +patchMergeKey=mountPath
	// +patchStrategy=merge
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty" patchStrategy:"merge" patchMergeKey:"mountPath"`
	// volumeDevices is the list of block devices to be used by the container.
	// +optional
	// +patchMergeKey=devicePath
	// +patchStrategy=merge
	VolumeDevices []corev1.VolumeDevice `json:"volumeDevices,omitempty" patchStrategy:"merge" patchMergeKey:"devicePath"`
	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`
	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
	// StartupProbe indicates that the Pod has successfully initialized.
	// If specified, no other probes are executed until this completes successfully.
	// If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	StartupProbe *corev1.Probe `json:"startupProbe,omitempty"`
	// Actions that the management system should take in response to container lifecycle events.
	// +optional
	Lifecycle *corev1.Lifecycle `json:"lifecycle,omitempty"`
	// Path at which the file to which the container's termination message
	// will be written is mounted into the container's filesystem.
	// Message written is intended to be brief final status, such as an assertion failure message.
	// Will be truncated by the node if greater than 4096 bytes.
	// Defaults to /dev/termination-log.
	// +optional
	TerminationMessagePath string `json:"terminationMessagePath,omitempty"`
	// Indicate how the termination message should be populated. File will use the contents of
	// terminationMessagePath to populate the container status message on both success and failure.
	// FallbackToLogsOnError will use the last chunk of container log output if the termination
	// message file is empty and the container exited with an error.
	// Defaults to File.
	// +optional
	TerminationMessagePolicy corev1.TerminationMessagePolicy `json:"terminationMessagePolicy,omitempty"`
	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// SecurityContext defines the security options the container should be run with.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
	// Whether this container should allocate a buffer for stdin in the container runtime.
	// If this is not set, reads from stdin in the container will always result in EOF.
	// Default is false.
	// +optional
	Stdin bool `json:"stdin,omitempty"`
	// Whether the container runtime should close the stdin channel after it has been opened by
	// a single attach. When stdin is true the stdin stream will remain open across multiple attach
	// sessions.
	// Default is false.
	// +optional
	StdinOnce bool `json:"stdinOnce,omitempty"`
	// Whether this container should allocate a TTY for itself, also requires 'stdin' to be true.
	// Default is false.
	// +optional
	TTY bool `json:"tty,omitempty"`
}

// AppClusterBindingStatus stores status information about an app cluster binding.
type AppClusterBindingStatus struct {
	// conditions is a list of conditions that apply to the AppClusterBinding.
	Conditions conditionsapi.Conditions `json:"conditions,omitempty"`
}

// AppClusterBindingList is a list of AppClusterBindings.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AppClusterBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AppClusterBinding `json:"items"`
}
