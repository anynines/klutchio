---
title: Developers Guide
sidebar_position: 5
tags:
  - Klutch
  - open source
  - application developer
keywords:
  - Klutch
  - open source
  - application developer
---

This guide helps you create and manage resources using Klutch, similar to how you interact with Kubernetes default
resources like deployments.

It assumes you have a [Klutch-enabled app cluster](./platform-operator-guide/setting-up-klutch-clusters/app-cluster.md).
If you don’t have a setup in place and want to experiment with Klutch in a local environment first, check out the [Local Deployment Guide](./local-deployment-guide.md)
for step-by-step instructions on deploying and testing Klutch locally.

## Prerequisites

Before you begin, make sure you have:

1. [kubectl](https://kubernetes.io/docs/tasks/tools/) installed and configured to interact with your App Cluster.
2. The connection between the App Cluster and the Klutch Control Plane Cluster has been successfully established. This
means the setup steps outlined in the [Klutch-enabled App Cluster](./platform-operator-guide/setting-up-klutch-clusters/app-cluster.md)
guide have been completed without issues.

## Creating a Resource

The types of resources you can create depend on the Klutch API bindings configured in your App Cluster. These may
include databases, message queues, storage solutions, and other services available through supported cloud providers or
on-premise infrastructure (automation backends).

Before creating a resource, you can check which APIs are available by listing the bound APIs. You can do this with the
following command:

```bash
kubectl get apiservicebindings
```

This will provide a list of the API bindings available in your cluster. However, it won’t give you details about the
structure of these APIs. To inspect a specific API binding in more detail, you can use:

```bash
kubectl describe crd <crd-name>
```

Replace \<crd-name> with the name of the apiservicebinding you are interested in.

If the CRD corresponding to an APIServiceBinding is missing, check whether the binding is healthy by inspecting its
status. An unhealthy binding may indicate that the associated CRD is unavailable.

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

## Managing Resources

### Checking Resource Status

To check the status of your resource, use the following commands:

```bash
kubectl get <resource_type> [resource_name]
```

```bash
kubectl describe <resource-type> [resource-name]
```

### Updating a Resource

There are two ways to update a resource:

1. Reapply the modified YAML file using the command:

    ```bash
    kubectl apply -f updated-resource.yaml
    ```

2. Make quick changes directly in the default editor (like Vim or Nano) using the command:

    ```bash
    kubectl edit <resource-type> [resource-name]
    ```

:::note
Some fields may be immutable once the resource is created. The specific immutable fields depend on the resource type.
:::

### Deleting Resources

To delete a resource, use the following command:

```bash
kubectl delete <resource-type> [resource-name]
```

This will trigger the deletion of the actual resource in the remote environment through Klutch.

## Troubleshooting

If you run into challenges while creating or managing resources, the following steps can help diagnose and possibly
resolve issues.

1. Check the resource status and events in your app cluster by using the following command:

    ```bash
    kubectl describe <resource-type> [resource-name]
    ```

2. Check the logs of the konnector (the Klutch component running in your App Cluster) to identify potential issues with
communication between your App Cluster and the Control Plane Cluster using the following command:

    ```bash
    kubectl logs -n kube-bind deployment/konnector
    ```

As a developer, your access is typically limited to your App Cluster. Troubleshooting issues related to the Klutch
Control Plane Cluster or Crossplane configuration may require additional permissions or support from your platform
operators.

For further questions or assistance, feel free to raise an issue on our [GitHub repository](https://github.com/anynines/klutchio),
or join our [Slack workspace](https://app.slack.com/client/T07FST6U1T7/C07FVLWBDDH). If you’re enjoying the software,
don’t forget to leave us a star on GitHub, it really makes our day!
