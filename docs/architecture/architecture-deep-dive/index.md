---
title: Architecture Deep Dive
tags:
  - Klutch
  - architecture  
  - open source
  - kubernetes
  - deep dive
keywords:
  - Klutch
  - architecture
  - open source
  - kubernetes
  - deep dive
---

In this section, we take a closer look at the architecture of Klutch, examining its key components and how they interact.
You will explore the components that make up the App Cluster and Klutch Control Plane Cluster, which facilitate data
service provisioning and management.

## App Cluster

![App Cluster](../../img/app-cluster.png)

### Konnector

The konnector is a lightweight Kubernetes deployment running in the App Cluster. After binding to a Klutch API in the
Control Plane Cluster, it ensures that the necessary Custom Resource Definitions (CRDs) are created in the App Cluster.
For more details on how APIs are made available to App Clusters and the API lifecycle in Klutch, see [this page](../../platform-operator-guide/integrating-ds-in-klutch.md#understanding-the-api-lifecycle-in-klutch).

In addition to managing CRDs, the konnector continuously synchronizes resources by propagating specifications from the
App Cluster to the Klutch Control Plane Cluster while retrieving and applying status updates from the Klutch Control
Plane Cluster back to the App Cluster.

### Klutch-bind CLI

The klutch-bind CLI (which is actually a kubectl plugin) runs on the developer's local machine and interacts directly
with the App Cluster. There are two main functions you can perform using the CLI:

1. Initiate the OpenID Connect (OIDC) authentication process and install the konnector into the App Cluster, which
happens during the first binding request.
2. Enable Klutch APIs across Kubernetes clusters by handling both authentication and the binding process.

## Control Plane Cluster

![Control Plane Cluster](../../img/control-plane-cluster.png)

### Klutch-bind backend

This component is triggered when the user initiates the authentication process from the App Cluster. It verifies the
user by sending a token (login credentials) to the OIDC provider. Once the user is authenticated, the component sets up
the necessary resources on the Control Plane Cluster.

### Crossplane provider(s)

Providers allow Crossplane to manage infrastructure on external services by introducing Kubernetes APIs that correspond
to the external service's APIs.

There are two provides that are currently supported by Klutch:

1. **Provider-anynines**: is a Crossplane provider that enables the creation, updating, and deletion of VM-based [a9s Data Service](https://www.anynines.com/data-services)
instances through Kubernetes resources.
2. **Provider-kubernetes**: is a Crossplane provider that enables the creation, updating, and deletion of
Kubernetes-based [a8s Data Service](https://k8s.anynines.com/for-postgres/)  through Kubernetes resources.

The two providers mentioned above highlight Klutch's platform-agnostic nature, showcasing its flexibility beyond
specific data services. By adding custom providers or configurations, you can [integrate any service](../../platform-operator-guide/integrating-ds-in-klutch.md)
hosted on any platform, extending the capabilities of Klutch to a wide range of use cases.

### Crossplane-api

This includes configuration files for all providers available in Klutch, such as Crossplane XRDs (Composite Resource
Definitions) and Crossplane Compositions, which define custom resource types and map them to actual infrastructure
resources across automation backends.

For more detailed information, you can check out the link [here](https://github.com/anynines/klutchio/tree/main/crossplane-api).

### Third-Party Components

#### Crossplane

Crossplane enables the management of infrastructure directly from Kubernetes by using providers, which are sets of
Custom Resource Definitions (CRDs) and their controllers that interface with external APIs. To learn more about
Crossplane, visit its [documentation](https://docs.crossplane.io/).

#### OIDC

Klutch uses a token-based authentication system powered by OpenID Connect (OIDC). The authentication process is
initiated through OIDC, with the Klutch-bind backend verifying the user's tokens via an OIDC provider. Currently,
Dex is the default OIDC provider, but other providers such as Keycloak can also be used.

While the implementation is still evolving, a commercial feature for Klutch will introduce enhanced authentication and
authorization capabilities in the future. For further details on OIDC-based authentication, please refer to the Klutch
Control Plane Cluster setup [page](../../platform-operator-guide/setting-up-klutch-clusters/control-plane-cluster/index.md#4-authentication-protocol-configuration).
