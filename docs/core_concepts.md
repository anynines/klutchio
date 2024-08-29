---
id: klutch-core-concepts
title: Core Concepts
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

- **Central Management Cluster:**
  - Hosts Crossplane and its providers
  - Runs the bind backend
- **Developer Cluster:**
  - Hosts klutch-bind CLI for creating bindings to remote resources
  - Hosts the konnector for state synchronization between the Custom Resources (CRs) in the central and developer cluster(s)

### 2. State Synchronization

The konnector component performs bidirectional state synchronization:

- Watches for changes in CRs on developer clusters
- Propagates these changes to the central management cluster
- Updates the status of resources in developer clusters based on the actual state in the central cluster

### 3. Authentication and Authorization

Klutch implements a token-based auth system:

- Uses OIDC for initial authentication
- The bind backend verifies tokens with the OIDC provider (e.g., Keycloak)

### 4. Remote / Proxy Resources

To manage remote resources, Klutch uses the concept of proxy resources:

- Proxy resources are CRs that represent remote resources in the central cluster
- Proxy resources map to [Crossplane Composite Resources (XRs)](https://docs.crossplane.io/master/concepts/composite-resources/)
- The resource management Klutch does is all based on the yaml files the user manages on their consumer clusters.
- The developers clusters are the source of truth for what resources should exist.

### 5. Binding Mechanism

Klutch's binding mechanism enables API usage across multiple Kubernetes clusters:

- **Klutch-bind CLI:** enables usage of Klutch's APIs across different Kubernetes clusters by synchronizing state
between the management cluster and API consumer clusters. The CLI
initiates the OIDC auth process and installs the konnector into the user's cluster.

- **bind backend:** The backend authenticates new users via OIDC before creating a binding space on the consumer cluster
for them. The backend implementation is open to different approaches, as long as they follow the standard.

- **konnector:** this component gets installed in the consumer's cluster and is responsible for synchronization between
the management cluster and the consumer cluster.

### 6. Provider Integration

Klutch leverages [Crossplane's provider](https://docs.crossplane.io/master/concepts/providers/) model:

- Supports any provider that adheres to Crossplane's provider specification
- Platform operators can install and configure providers in the central cluster
- Providers handle the actual interaction with cloud APIs or infrastructure management tools
