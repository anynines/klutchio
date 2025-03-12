---
title: Integrating Data Services in Klutch
sidebar_position: 4
tags:
  - Klutch
  - Kubernetes
  - data service integration
keywords:
  - Klutch
  - Kubernetes
  - data service integration
---

As a platform operator, you may need to introduce internal or specialized data services to your Klutch environment,
distributing them similarly to officially supported services. This approach is beneficial for customization,
white-labeling, or incorporating data services not natively supported by Klutch. This guide outlines the process of
creating and integrating custom data services into Klutch.

## Adding a new Data Service in Klutch

To introduce a new data service in Klutch, follow these steps:

1. **Define the API**: Create the necessary [Crossplane Composite Resource Definitions](https://docs.crossplane.io/latest/concepts/composite-resource-definitions/)
(XRDs) and [Compositions](https://docs.crossplane.io/latest/concepts/compositions/) for your data service. You can find
examples of this in the [Klutch repository](https://github.com/anynines/klutchio/tree/main/crossplane-api/api/a9s).
2. **Installation**: Build a Crossplane configuration package and push it to your chosen OCI-compliant image registry,
then deploy the configuration package. More detailed steps are described in the [Klutch repository](https://github.com/anynines/klutchio/tree/main/crossplane-api).
Alternatively, you can directly apply the Crossplane Composition and XRD files to the Klutch Control Plane cluster.
3. **Expose the API via Klutch**: Make the new API available for binding to App clusters by creating an
APIServiceExportTemplate like this:

    ```yaml
    apiVersion: example-backend.klutch.anynines.com/v1alpha1
    kind: APIServiceExportTemplate
    metadata:
        name: <descriptive-name>
        namespace: crossplane-system
    spec:
        APIServiceSelector:
            group: <api-group>
            resource: <resource-name-plural>
            version: <resource-version>
    ```

    Replace the placeholders with your service-specific details:
    - **\<descriptive-name>**: A unique identifier for your service.
    - **\<api-group>**: The API group of your service.
    - **\<resource-name-plural>**: The plural name of your resource.
    - **\<resource-version>**: The version of your resource.

    Applying this custom resource to the Klutch Control Plane cluster will make your API available for binding using the
    web interface. If your service requires additional resources (e.g., Secrets, ConfigMaps) to be synchronized between
    clusters, specify these in the permissionClaims section of the APIServiceExportTemplate.

    The example below shows the servicebindings API shared via klutch-bind, with the additional permission claims to
    synchronize secrets and config maps from the Control Plane cluster to the app cluster. Syncing of claimed resources
    always includes all resources of that type in all bound namespaces.

    ```yaml
    kind: APIServiceExportTemplate
    apiVersion: example-backend.klutch.anynines.com/v1alpha1
    metadata:
        name: "servicebindings"
        namespace: crossplane-system
    spec:
        APIServiceSelector:
            resource: servicebindings
            group: anynines.com
        permissionClaims:
            - group: ""
              resource: secrets
              version: v1
              selector:
                owner: Provider
            - group: ""
              resource: configmaps
              version: v1
              selector:
                owner: Provider
    ```

    :::info
    The servicebindings API used in this example is specific to Klutch and is not related to the [Service Binding Specification for Kubernetes](https://servicebinding.io/). 
    While both aim to streamline application connectivity to services, their approaches vary.
    :::

## Modifying an Existing Klutch Data Service

As a platform operator, you may need to modify an existing Klutch data service API to better align with your
organization's requirements. This involves updating the XRDs and/or Composition files. After implementing the desired
changes, you can deploy the updated configuration to the Control Plane Cluster.

Here's how to proceed:

1. **Retrieve the Existing Files**: Obtain the current Crossplane XRD and/or Composition files that correspond to the
data service API you want to modify.

2. **Implement Modifications**: Modify the files as needed, such as updating service plans or renaming APIs to reflect
your branding.

3. **Deploy the Updated Configuration**: Build a Crossplane configuration package and push it to your chosen
OCI-compliant image registry. Then, deploy the updated configuration package to the Control Plane Cluster. Detailed
steps are available in the [Klutch crossplane-api repository](https://github.com/anynines/klutchio/tree/main/crossplane-api).

## Understanding the API Lifecycle in Klutch

Understanding the journey of an API through Klutch stack is crucial for effective management and integration.

1. **API Definition**: The process starts by defining the API in a Crossplane Configuration Package. When installed in
the Control Plane Cluster, Crossplane extracts the API definitions.

2. **Making APIs Available to App Clusters**: To share the resource with App Clusters, the platform operator creates an
APIServiceExportTemplate. When an API binding is initiated, the App Cluster generates an APIServiceExportRequest
on the Control Plane Cluster.

3. **Permission Granting and Export Creation**: Upon the creation of the APIServiceExportRequest, the Klutch-bind backend
assigns the necessary permissions to the App Cluster's Kubernetes service account for interacting with the requested API
and its associated objects. Subsequently, the Klutch-bind backend creates an APIServiceExport object containing a snapshot
of the bound Custom Resource Definition (CRD) at the time of binding.

4. **API Binding Process**: The application developer applies an APIServiceBinding object to their cluster, typically
executed via the kubectl-bind command. The konnector, installed in the App Cluster, detects this event, reads the
APIServiceBinding object, and searches for a corresponding APIServiceExport on the Control Plane Cluster. If a match
is found, the konnector retrieves the API schema from the APIServiceExport and creates a CRD with a matching schema
on the App Cluster. This continuous process accommodates changes and additions of new APIs as they occur.
