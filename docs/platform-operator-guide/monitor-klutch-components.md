---
title: Monitoring Klutch Components
sidebar_position: 2
tags:
  - Klutch
  - Kubernetes
  - monitoring
  - observability
keywords:
  - Klutch
  - Kubernetes
  - monitoring
  - observability
---

As a platform operator, maintaining the health and performance of Klutch's components is essential. This guide provides
key insights into effective monitoring.

## Control Plane Cluster Monitoring

### Crossplane® Providers Health

Klutch utilizes Crossplane® providers, such as provider-anynines and provider-kubernetes. To assess their health,
verify the HEALTHY status of the installed providers:

```bash
kubectl get providers
```

Example output:

```bash
NAME                                     INSTALLED   HEALTHY   PACKAGE                                                         AGE
provider-anynines                        True        True      public.ecr.aws/w5n9a2g2/klutch/provider-anynines:v1.3.2         10m
```

A `True` value under the HEALTHY column indicates a properly functioning provider. If a provider's pod encounters an
issue and restarts, the HEALTHY status may temporarily display `false`. Often, the system resolves this automatically.

For persistent issues, examine the logs of the affected pods. Identify these pods using the `pkg.crossplane.io/provider`
label.

*Example: Locating provider-anynines pods:*

```bash
kubectl get pods --namespace crossplane-system -l pkg.crossplane.io/provider=provider-anynines
```

### Monitoring provider-anynines Configuration

The provider-anynines Crossplane® provider connects to data service brokers using a ProviderConfig. It includes a
health probe that periodically checks the connection and broker availability, with the results reflected in the status
of each ProviderConfig.

To check the health status, list the provider configurations:

```bash
kubectl get providerconfigs
```

Example output:

```bash
NAME                        AGE   HEALTHY
postgresql-backup-manager   10m   true
postgresql-service-broker   10m   true
```

Each entry shows a HEALTHY status. For detailed information, inspect the `status.health` field of a specific
configuration:

```bash
kubectl get providerconfigs.dataservices.anynines.com <config-name> -o yaml
```

Replace \<config-name> with the name of your configuration. A failed health check sets `lastStatus` to `false`, with
`lastMessage` providing details about the failure.

### HTTP-Based Health Monitoring

For integration with monitoring systems, provider-anynines offers an HTTP endpoint at `/backend-status` on port 8081,
summarizing the health of all ProviderConfigs. To access this endpoint within the Control Plane Cluster, create a
service:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: provider-anynines-health
  namespace: crossplane-system
spec:
  selector:
    pkg.crossplane.io/provider: provider-anynines
  ports:
    - port: 8081
      protocol: TCP
```

This service can be exposed externally using standard Kubernetes methods, such as LoadBalancer or Ingress.

*Example: Accessing Health Endpoint via HTTP*

```bash
curl -i http://provider-anynines-health:8081/backend-health
```

A successful response indicates the health status of your ProviderConfig. Below are examples of a healthy and an
unhealthy response:

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs>
  <TabItem value="Healthy" label="Healthy" default>

  ```json
  HTTP/1.1 200 OK
  Content-Type: application/json
  Date: Tue, 20 Aug 2024 13:14:57 GMT
  Content-Length: 80

  {
    "postgresql-backup-manager": "ok",
    "postgresql-service-broker": "ok"
  }
  ```

  </TabItem>

  <TabItem value="Unhealthy" label="Unhealthy">

  ```json
  HTTP/1.1 200 OK
  Content-Type: application/json
  Date: Tue, 20 Aug 2024 13:14:57 GMT
  Content-Length: 80

  {
    "postgresql-backup-manager": "ok",
    "postgresql-service-broker": "degraded"
  }
  ```

  </TabItem>
</Tabs>

### Monitoring Klutch-bind backend

The klutch-bind backend includes Kubernetes-level health checks. To monitor its health, observe the `Available`
condition of its deployment:

```bash
kubectl get deployments anynines-backend --namespace bind
```

Example output:

```bash
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
anynines-backend   1/1     1            1           10m
```

An `AVAILABLE` status confirms the backend is operational.

## App Cluster Monitoring

### Monitoring konnector Deployment

The konnector includes Kubernetes health checks. To monitor its health, observe the `Available` condition of its
deployment:

```bash
kubectl get deployments konnector --namespace bind
```

Example output:

```bash
NAME        READY   UP-TO-DATE   AVAILABLE   AGE
konnector   2/2     2            2           10m
```

An `AVAILABLE` status indicates the konnector is functioning correctly.
