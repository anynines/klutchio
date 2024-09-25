---
title: Configuring Crossplane Provider provider-anynines
sidebar_position: 2
tags:
  - control plane cluster
  - kubernetes
  - a9s data services
  - platform operator
keywords:
  - a9s data services
  - platform operator
---

### Prerequisites

If you want to use [a9s Data Services](https://www.anynines.com/data-services) in conjunction with 
Klutch, you can use `provider-anynines` to talk to the service broker.

In order to follow along with this manual, you need a working installation of the CloudFoundry 
service broker and a pair of credentials. The service broker must be reachable from the network of 
the Control Plane Cluster's worker nodes.

### Install ProviderConfig

To configure the Crossplane provider `provider-anynines`, you will need to update and apply the
following YAML file for each a9s Data Service you want to be able to use. Replace the
`<data-service>` placeholder in the following YAML file with the corresponding value from the table
below for the Data Service you want to deploy:

| Data Service   | Data-service Value |
| -------------- | ------------------ |
| a9s Redis      | redis              |
| a9s Messaging  | messaging          |
| a9s Logme2     | logme2             |
| a9s Prometheus | prometheus         |
| a9s Search     | search             |
| a9s MongoDB    | mongodb            |
| a9s MariaDB    | mariadb            |
| a9s PostgreSQL | postgresql         |

Additionally, substitute the remaining placeholder values denoted by `< >` with the actual variable
values, as described for each Data Service you want to support.

After making these updates, apply the modified YAML file to enact the changes.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <data-service>-service-broker-creds
  namespace: crossplane-system
type: Opaque
data:
  username: <service-broker-username-base64-encoded>
  password: <service-broker-password-base64-encoded>
---
apiVersion: dataservices.anynines.com/v1
kind: ProviderConfig
metadata:
  name: <data-service>-service-broker
spec:
  url: <service-broker-url> # e.g. http://example.com:3000
  providerCredentials:
    source: Secret
    username:
      secretRef:
        namespace: crossplane-system
        name: <data-service>-service-broker-creds
        key: username
    password:
      secretRef:
        namespace: crossplane-system
        name: <data-service>-service-broker-creds
        key: password
---
apiVersion: v1
kind: Secret
metadata:
  name: <data-service>-backup-manager-creds
  namespace: crossplane-system
type: Opaque
data:
  username: <backup-manager-username-base64-encoded>
  password: <backup-manager-password-base64-encoded>
---
apiVersion: dataservices.anynines.com/v1
kind: ProviderConfig
metadata:
  name: <data-service>-backup-manager
spec:
  url: <backup-manager-url> # e.g. http://example.com:3000
  providerCredentials:
    source: Secret
    username:
      secretRef:
        namespace: crossplane-system
        name: <data-service>-backup-manager-creds
        key: username
    password:
      secretRef:
        namespace: crossplane-system
        name: <data-service>-backup-manager-creds
        key: password
```

<a href="/po_files/providerconfig.yaml" target="_blank" download>Download providerconfig.yaml</a>

To verify that the providerconfigs are correct, check their status and wait for them to all be
"healthy":

```bash
$ kubectl get providerconfigs
NAME                        AGE     HEALTHY
postgresql-backup-manager   10s     true
postgresql-service-broker   10s     true
...
```
