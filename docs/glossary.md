---
title: Glossary
sidebar_position: 7
tags:
  - Klutch
  - open source
  - kubernetes
  - glossary
keywords:
  - Klutch
  - open source
  - kubernetes
  - glossary
---

This page explains key terms and concepts to help contributors and users better navigate our documentation and codebase.

#### App Cluster

An App Cluster is a Kubernetes cluster that is used by application developers to consume Klutch Resources. For example,
an App Cluster may host an application that utilizes a PostgreSQL database provisioned by Klutch. App Clusters are never
the environment where the actual data services are provisioned

#### Klutch Control Plane Cluster

Klutch Control Plane Cluster is a Kubernetes cluster that oversees the entire multi-cluster ecosystem. It manages
bidirectional synchronization of resource specifications, status, and additional information with App Clusters, while
maintaining a catalog of available resources. This cluster provides system-wide observability and management
capabilities. It hosts Crossplane and its providers to manage and provision cloud-native resources across multiple
automation backends.

#### Proxy Claim

A proxy claim is a Kubernetes Custom Resource (CR) that developers create in the App Cluster. It acts as a proxy for
[Crossplane Resource Claim](https://docs.crossplane.io/latest/concepts/claims/) (XRC) requests, which originate from the
Control Plane cluster and trigger the provisioning of actual resources in an automation backend. Klutch handles the
synchronization of these claims, ensuring seamless integration and resource management.

#### Service Binding

A service binding connects your application to a data service instance. When you create a service binding resource in
your App Cluster, Klutch generates a secret containing all the necessary information for this connection. Your
application can then use this secret to connect to the data service instance.

:::note

Klutch's Service Binding differs from the [Service Binding Specification for Kubernetes](https://github.com/servicebinding).
While both aim to streamline application connectivity to services, their approaches vary.

:::

#### API Binding

API Binding refers to the process of establishing a connection between an App Cluster and Klutch's APIs in the Control
Plane cluster. This enables the App Clusters to interact with and leverage the resources and services managed by Klutch.

#### Data Services

Data services refer to software or platform components responsible for managing, storing, processing, or transporting
data, such as databases, message brokers, and similar technologies.

#### Automation Backends

Automation backends are responsible for performing the actual resource provisioning, whether for virtual machines (VMs),
containers, cloud infrastructure, or other resources, based on the configuration defined within Klutch. Automation
backends in Klutch can include Service Brokers, Kubernetes Operators, and vendor-specific APIs (such as AWS, Azure,
and GCP).
