---
id: a9s-po-monitoring
title: Monitoring
---

# Monitoring

The platform components contain facilities to monitor their health. This page describes these
facilities and how to use them.

## Central Management Cluster

### Crossplane providers

The health of both providers ("anynines" and "kubernetes") can be observed by checking the "HEALTHY"
condition of the installed crossplane providers:

```
$ kubectl get providers
NAME                                     INSTALLED   HEALTHY   PACKAGE                                                         AGE
crossplane-contrib-provider-kubernetes   True        True      xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.9.0   118m
provider-anynines                        True        True      public.ecr.aws/w5n9a2g2/anynines/provider-anynines:v1.2.0       118m
```

If one of the underlying pods encounters an error and needs to be recreated, the HEALTHY condition
will temporarily become `false`. In many cases the situation will resolve itself automatically. To
troubleshoot any issues, check the logs of the relevant pods. These pods can be identified by the
`pkg.crossplane.io/provider` label.

Example for finding `provider-anynines` pods:

```
$ kubectl get pods --namespace crossplane-system -l pkg.crossplane.io/provider=provider-anynines
...
```

Example for finding `provider-kubernetes` pods:

```
$ kubectl get pods --namespace crossplane-system -l pkg.crossplane.io/provider=provider-kubernetes
...
```

### provider-anynines configuration health

Crossplane provider `provider-anynines` uses `ProviderConfig` resources to establish its connections
with the data service brokers. To ensure that the data service brokers are reachable, and that the
configured credentials work, the provider contains a health probe which periodically checks if each
of the `ProviderConfig`s is correct and if the respective service broker is available. The result of
these checks is reflected within the `status` of each `ProviderConfig` resource.

#### Observing health on ProviderConfig resource

To observe the result of the last health check, list the provider configs:

```
$ kubectl get providerconfigs
NAME                        AGE   HEALTHY
postgresql-backup-manager   10m   true
postgresql-service-broker   10m   true
...
```

:::note

Every crossplane provider has a different Custom Resource Definition (CRD) for its configurations.
Conventionally these are called "providerconfigs". Thus, there can be multiple resources with the
name "providerconfigs" defined in your cluster. To avoid any name clashes, you can always refer to
the custom resource of the configuration of `provider-anynines` by its full name
`providerconfigs.dataservices.anynines.com`.

Example:

```
kubectl get providerconfigs.dataservices.anynines.com
```

For brevity though, throughout this documentation we stick with the short name.

:::

For more detailed information, inspect the `status.health` field of each config.

Example:

```
$ kubectl get providerconfigs postgresql-backup-manager -oyaml
...
status:
  health:
    lastCheckTime: "2024-08-20T12:11:25Z"
    lastMessage: Available
    lastStatus: true
```

In case of a failed health check, `lastStatus` is set to `false` and `lastMessage` should contain
details about the cause of the failure.

#### Observing health via HTTP

For easier integration in monitoring systems, the anynines provider exposes an HTTP endpoint
accumulating the healthiness of all `ProviderConfigs`. The endpoint is called `/backend-status` and
is reachable on port **8081** of the `provider-anynines` pods. By default the endpoint is not
exposed in any way. To make it reachable from inside the central management cluster, create a
service such as this:

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

This service can be made accessible from outside of the cluster, by the usual kubernetes means (e.g.
LoadBalancer service, Ingress, ...).

##### Example request via HTTP (all healthy)

```
$ curl -i http://provider-anynines-health:8081/backend-health
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 20 Aug 2024 13:14:57 GMT
Content-Length: 80

{"postgresql-backup-manager":"ok","postgresql-service-broker":"ok"}
```

##### Example request via HTTP (one of the configurations is unhealthy)

```
$ curl -i http://provider-anynines-health:8081/backend-health
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 20 Aug 2024 13:14:57 GMT
Content-Length: 80

{"postgresql-backup-manager":"ok","postgresql-service-broker":"degraded"}
```

### klutch-bind backend

The klutch-bind backend includes kubernetes level health checks. To monitor it's healthiness,
observe the "Available" condition of it's deployment:

```
$ kubectl get deployments anynines-backend --namespace bind
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
anynines-backend   1/1     1            1           1h
```

## Developer cluster

### Konnector

The konnector includes Kubernetes health checks. To monitor it's healthiness, observe the
"Available" condition of it's deployment:

```
$ kubectl get deployments konnector --namespace bind
NAME        READY   UP-TO-DATE   AVAILABLE   AGE
konnector   2/2     2            2           1h
```
