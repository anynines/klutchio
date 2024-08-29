---
id: klutch-developers
title: For developers
tags:
  - Klutch
  - open source
  - kubernetes
  - application developer
keywords:
  - Klutch
  - open source
  - kubernetes
  - application developer
---

## Creating and Managing Resources

This section explains how to create and manage remote resources using Klutch. As a developer, you can provision and
manage cloud or on-premise resources directly from your Kubernetes cluster using a declarative approach.

![Developer Interaction with Klutch](./developer_interactions.svg)

## Prerequisites

Before you begin, ensure you have:

1. [kubectl](https://kubernetes.io/docs/tasks/tools/) installed and configured to interact with your developer cluster.
2. Access to a Klutch-enabled Kubernetes cluster.

   If your cluster wasn't set up by a platform operator, you need to use the klutch-bind CLI to connect to the central
   management cluster and bind to the resources you intend to use. For instructions on using the klutch-bind CLI, refer
   to the ["For Platform Operators"](../platform-operator/index.md) section.

## Available Resource Types

The types of resources you can create depend on the service bindings configured in your developer cluster. These can
include databases, message queues, storage solutions, or any other services available in supported cloud providers or
on-premise infrastructure.

These resources are represented as standard [Kubernetes Custom Resources (CRs)](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/).
You can list the supported Custom Resource Definitions (CRDs) just as you would with any other Kubernetes CRs using the
following command:

```bash
kubectl get crds
```

To view the schema of a specific CRD:

```bash
kubectl explain <crd-name>
```

For a detailed view of the CRD's structure:

```bash
kubectl get crd <crd-name> -o yaml
```

## Creating a Resource

To create a resource, define it using a Custom Resource (CR) YAML file and apply it to your cluster.

1. Create a YAML file for your resource. Here's a generic template:

    ```yaml
    apiVersion: <resource-group>/<version>
    kind: <ResourceType>
    metadata:
      name: my-resource
    spec:
      # Resource-specific fields go here
    ```

2. Apply the YAML file to your cluster:

    ```bash
    kubectl apply -f my-resource.yaml
    ```

3. Monitor the resource creation:

    ```bash
    kubectl get <resource-type> my-resource -w
    ```

    This command watches the resource and displays status updates. It shows whether the Custom Resource is successfully
    synced with the remote resource and indicates when the remote resource is ready for use.

## Managing Resources

### Checking Resource Status

To list resources of a specific type:

```bash
kubectl get <resource-type>
```

For detailed information about a specific resource:

```bash
kubectl describe <resource-type> <resource-name>
```

### Updating Resources

To update a resource, modify its YAML file and reapply it:

```bash
kubectl apply -f updated-resource.yaml
```

:::note

Some fields may be immutable once the resource is created. The specific immutable fields depend on the resource type.

:::

### Deleting Resources

To delete a resource:

```bash
kubectl delete <resource-type> <resource-name>
```

This will trigger the deletion of the actual resource in the remote environment through Crossplane.

## Troubleshooting

If you encounter issues while creating or managing resources, the following steps can help diagnose and potentially
resolve the problem. Depending on your specific situation and level of access, some or all of these steps may be
applicable:

1. Check the resource status and events in your developer cluster:

   ```bash
   kubectl describe <resource-type> <resource-name>
   ```

   Look for events or status messages that might indicate the issue.

2. Examine the logs of konnector (the component of Klutch running in your developer cluster):

   ```bash
   kubectl logs -n klutch-bind deployment/konnector
   ```

   This may show issues related to the communication between your developer cluster and the central management cluster.

3. If you have access to the central management cluster and are familiar with the Crossplane setup and configuration,
   you can perform additional troubleshooting steps in the central management cluster. Refer to the latest [official Crossplane troubleshooting guide](https://docs.crossplane.io/latest/guides/troubleshoot-crossplane/) for comprehensive instructions.

   Some key steps you can take include:

   a. Verify the status of the corresponding Crossplane XR (Composite Resource):

      ```bash
      kubectl get <xr-type> <xr-name> -n <namespace>
      ```

   b. Check the status of the Crossplane controllers:

      ```bash
      kubectl get pods -n crossplane-system
      ```

      Ensure all pods are in the `Running` state.

   c. Examine Crossplane controller logs:

      ```bash
      kubectl logs -n crossplane-system <crossplane-pod-name>
      ```

      Look for any error messages related to your resource.

    d. Verify that the provider for your resource is properly installed:

      ```bash
      kubectl get providers
      ```

   e. Inspect managed resources:

      ```bash
      kubectl get managed
      ```

      Identify any resources in a non-ready state.

   f. Check package installation status (if using Crossplane's package manager):

      ```bash
      kubectl get packages
      ```

Remember that as a developer, your access is typically limited to your developer cluster. Many advanced troubleshooting
steps, especially those involving the central management cluster or Crossplane configuration, may require collaboration
with your platform operators or additional permissions. If you suspect a bug in Klutch, please consider opening an issue
in the relevant GitHub repository with a detailed description of the problem and steps to reproduce it.
