---
title: Core Concepts
sidebar_position: 2
tags:
  - Klutch
  - open source
  - kubernetes
  - core concepts
keywords:
  - Klutch
  - open source
  - kubernetes
  - core concepts
---

Klutch extends Crossplane to manage resources across multiple Kubernetes clusters. The following technical concepts are
fundamental to understanding Klutch's architecture and operation:

![Klutch detailed architecture diagram](<klutch_components.svg>)

### 1. Cluster Topology

Klutch operates on a multi-cluster model:

- **Control Plane Cluster:**
  - Hosts Crossplane and its providers
  - Runs the bind backend
- **App Cluster:**
  - Hosts klutch-bind CLI for creating bindings to remote resources
  - Hosts the konnector for state synchronization between the Custom Resources (CRs) in the Control Plane and App cluster(s)

### 2. State Synchronization

The konnector component performs bidirectional state synchronization:

- Watches for changes in CRs on App Clusters
- Propagates these changes to the Control Plane Cluster
- Updates the status of resources in App Clusters based on the actual state in the Control Plane Cluster

### 3. Authentication and Authorization

Klutch implements a token-based auth system:

- Uses OIDC for initial authentication
- The bind backend verifies tokens with the OIDC provider (e.g., Keycloak)

### 4. Proxy Claims

To manage remote resources, Klutch uses the concept of Proxy Claims:

- Proxy Claims are applied in the App Cluster.
- These Proxy Claims are mapped to [Crossplane Composite Resources Claims (XRCs)](https://docs.crossplane.io/latest/concepts/claims/)
  in the Control Plane Cluster.
- The developers clusters are the source of truth for what resources should exist.

### 5. Binding Mechanism

Klutch's binding mechanism enables API usage across multiple Kubernetes clusters:

- **Klutch-bind CLI:** enables usage of Klutch's APIs across different Kubernetes clusters by synchronizing state
between the Control Plane Cluster and API App Clusters. The CLI initiates the OIDC auth process and installs the
konnector into the user's cluster.

- **bind backend:** The backend authenticates new users via OIDC before creating a binding space on the App Cluster
for them. The backend implementation is open to different approaches, as long as they follow the standard.

- **konnector:** this component gets installed in the App Cluster and is responsible for synchronization between
the Control Plane Cluster and the App Cluster.

### 6. Provider Integration

Klutch leverages [Crossplane's provider](https://docs.crossplane.io/master/concepts/providers/) model:

- Supports any provider that adheres to Crossplane's provider specification
- Platform operators can install and configure providers in the Control Plane Cluster
- Providers handle the actual interaction with cloud APIs or infrastructure management tools
