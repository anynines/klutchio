---
title: Introduction
sidebar_position: 1
tags:
  - klutch
  - documentation
  - open source
  - kubernetes
keywords:
  - klutch
  - documentation
  - open source
  - kubernetes
---

## Overview

Klutch is a Kubernetes-native platform that simplifies the orchestration of resources and data services across diverse cloud
environments and fleets of Kubernetes clusters. It enables on-the-fly provisioning of services by multiple application
Kubernetes clusters using a declarative interface. It caters to both the needs of platform operators and application
developers, as it simplifies both the hosting as well as the consumption of data services.

### Key Features

- **Multi-cluster Management**: Orchestrate resources and data services across fleets of Kubernetes clusters.
- **On-demand Provisioning**: Allow App Clusters to provision services dynamically as needed.
- **Declarative Interface**: Utilize a unified Kubernetes-native declarative approach for data service provisioning and
  consumption.
- **Unified Control**: Manage resources and data services across multiple environments from a single point.
- **Dual-focused Design**: Simplify operations for both platform operators and application developers.
- **Extensible Architecture**: Plugin-based architecture facilitates easy integration of new resource types and cloud
  providers.

### Architecture Overview

![Architecture overview](./architecture_overview.svg)

#### Overview of System Architecture and Developer/Operator Interactions

1. **Developers**

    - Interact with their Kubernetes clusters to request and use remote resources.
    - Define data service requirements using Custom Resources, initiating automated provisioning and deployment processes.

2. **App Clusters**

    - Request services from the Control Plane Cluster.
    - Utilize Klutch-bind to subscribe and enable usage of Klutch's APIs across different Kubernetes clusters.
    - Synchronize resource specifications, status, and additional information with the Control Plane Cluster.

3. **Platform Operators**

    - Configure and manage available remote resources through the Control Plane Cluster.
    - Oversee the entire system, ensuring smooth operation.

4. **Control Plane Clusters**

    - Manage the entire ecosystem using the centralized control plane.
    - Process service requests from App Clusters.
    - Utilize Crossplane for managing and provisioning cloud-native resources across multiple cloud providers and
    on-premise environments.
    - Manage bidirectional synchronization of resource specifications, status, and additional information with App
    Clusters.
    - Key functionalities include:
        - Maintain a list of available resources.
        - Handle the actual provisioning of resources in target environments.
        - Provide system-wide observability.

5. **Cloud Providers & On-Premise Infrastructure**

    - Serve as the actual environments where resources are provisioned.
    - Support hybrid and multi-cloud deployment models.

:::note

For a detailed breakdown of components and their interactions, please refer to the "Core Concepts" section.

:::
