---
title: Setting up Klutch Clusters
sidebar_position: 1
tags:
  - Klutch
  - Kubernetes
  - cluster setup
  - control plane cluster
  - app cluster
  - platform operator
keywords:
  - Klutch
  - Kubernetes
  - cluster setup
  - control plane cluster
  - app cluster
  - platform operator
---

Klutch requires two types of Kubernetes clusters: a **Control Plane Cluster** and one or more **App Clusters**. The
Control Plane Cluster serves as the central management layer for data services, while App Clusters connect to it,
enabling developers to provision and use data services.

Start by deploying and configuring the Control Plane Cluster. Once it’s operational, set up your App Clusters and
establish connections between them and the Control Plane Cluster. The [Setting Up the Klutch Control Plane Cluster](./control-plane-cluster/index.md)
and [Setting Up App Clusters](./app-cluster.md) sections provide step-by-step instructions for configuring these
clusters.

:::info Need Help?

For further information, troubleshooting, and discussions, visit our [GitHub repository](https://github.com/anynines/klutchio),
where you can ask questions, report issues, and contribute. 🤝

:::
