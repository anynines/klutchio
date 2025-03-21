---
title: Updating Klutch Components
sidebar_position: 3
tags:
  - Klutch
  - Kubernetes
  - updates
  - upgrades
  - maintenance
keywords:
  - Klutch
  - Kubernetes
  - updates
  - upgrades
  - maintenance
---

Regular updates to Klutch's components are essential for maintaining optimal performance, security, and access to new
features. This guide provides platform operators with detailed instructions for updating both the Control Plane and App
Cluster components within a Klutch environment.

## Updating the Control Plane Cluster

### Crossplane® Runtime

To update the Crossplane® runtime, refer to the [official Crossplane® documentation](https://docs.crossplane.io/). The
specific update process varies based on your installation method.

### Crossplane® Providers and Configuration Packages

Patching is a method for updating Crossplane® providers and configuration packages dynamically without modifying full
manifests.

You can see an example of updating a provider and its configuration package using patching with provider-anynines:

1. **Updating a Provider**

    As an example, to update provider-anynines, identify the latest version from the [image registry](https://gallery.ecr.aws/w5n9a2g2/klutch/provider-anynines),
    then apply the update:

    ```bash
    kubectl patch providers/provider-anynines --type merge -p '{"spec":{"package":"public.ecr.aws/w5n9a2g2/klutch/provider-anynines:<latest-version>"}}'
    ```

    Replace \<latest-version> with the desired release.

2. **Updating a Configuration Package**

    Similarly, to update a configuration package, you can use provider-anynines configuration as an example:

    ```bash
    kubectl patch configurations/anynines-dataservices \
    --type merge -p '{"spec":{"package":"public.ecr.aws/w5n9a2g2/klutch/dataservices:<latest-version>"}}'
    ```

    Ensure \<latest-version> matches the desired release.

### Klutch-bind backend

:::caution

Before proceeding, review the changelog for the new version and follow any migration instructions provided.

:::

1. **Install Latest CRDs**
    
    Update the Custom Resource Definitions (CRDs) for the backend as per the [backend installation instructions](../platform-operator-guide/setting-up-klutch-clusters/control-plane-cluster/index.md#3-deploy-klutch-bind-backend).

2. **Update Klutch-bind backend Deployment**
    
    Follow the [installation instructions](../platform-operator-guide/setting-up-klutch-clusters/control-plane-cluster/index.md#3-deploy-klutch-bind-backend)
    to update the Klutch-bind backend deployment.

3. **Introduce New Data Service Types**

    If the update includes new data service types, implement the [binding creation steps](https://klutch.io/docs/platform-operator/binding-creation)
    to make them available in App Clusters.

**Downtime Considerations During Updates**

During updates, components like Crossplane® providers and the Klutch-bind backend may experience brief downtime. However,
existing data service instances will remain operational and accessible. Any modifications (creation, updates, deletions)
made to data service instances during the update will be applied once the process is complete.

## Updating the App Cluster

The konnector deployment is the primary component of the App Cluster. To update it, first, retrieve the latest konnector
image from the [image registry](https://gallery.ecr.aws/w5n9a2g2/klutch/konnector) and replace \<latest-version> with
the desired version number. Then, choose one of the following update methods:

1. **Patching the Deployment**

    Apply a patch to update the konnector image:

    ```bash
    kubectl set image --namespace kube-bind deployment/konnector konnector=public.ecr.aws/w5n9a2g2/klutch/konnector:<latest-version>
    ```

2. **Using a Manifest**

    Apply an updated deployment manifest:

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
    name: konnector
    namespace: kube-bind
    labels:
        app: konnector
    spec:
    replicas: 2
    selector:
        matchLabels:
        app: konnector
    template:
        metadata:
        labels:
            app: konnector
        spec:
        restartPolicy: Always
        serviceAccountName: konnector
        containers:
            - name: konnector
            image: public.ecr.aws/w5n9a2g2/klutch/konnector:<latest-version>
            env:
                - name: POD_NAME
                valueFrom:
                    fieldRef:
                    fieldPath: metadata.name
                - name: POD_NAMESPACE
                valueFrom:
                    fieldRef:
                    fieldPath: metadata.namespace
    ```
