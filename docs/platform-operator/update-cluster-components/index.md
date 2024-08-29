---
id: klutch-po-update-cluster-components
title: Updating cluster components
tags:
  - tenant cluster
  - kubernetes
  - a9s data services
  - platform operator
keywords:
  - a9s data services
  - platform operator
---

This page documents the update process for all Klutch components.

## Updating tenant cluster

The tenant cluster contains only one component: the `konnector` deployment. To update the
`konnector`, simply change it's container image to the new one. The latest image can be found by
checking out the tab "Image tags" for this image in our
[image registry](https://gallery.ecr.aws/w5n9a2g2/anynines/konnector).

### Example using kubectl

```bash
kubectl set image --namespace bind deployment/konnector konnector=public.ecr.aws/w5n9a2g2/anynines/konnector:v1.3.0
```

### Example using a manifest

Apply this updated deployment manifest:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: konnector
  namespace: bind
  labels:
    app: konnector
spec:
  replicas: 2
  selector:
    matchLabels:
      app: konnector
  template:
    metadata:
      labels:
        app: konnector
    spec:
      restartPolicy: Always
      serviceAccountName: konnector
      containers:
        - name: konnector
          # This image should point to the new version:
          image: public.ecr.aws/w5n9a2g2/anynines/konnector:v1.3.0
          env:
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
```

## Updating provider cluster

### Crossplane runtime

To update the crossplane runtime, please refer to the
[upstream documentation](https://docs.crossplane.io/v1.14/software/install/). The exact process will
depend on your installation method.

### Crossplane providers and configuration packages

1. Update provider-anynines

The latest `provider-anynines` image can be found by checking out the tab "Image tags" for this
image in our [image registry](https://gallery.ecr.aws/w5n9a2g2/anynines/provider-anynines).

```bash
kubectl patch providers/provider-anynines \
  --type merge -p '{"spec":{"package":"public.ecr.aws/w5n9a2g2/anynines/provider-anynines:v1.3.0"}}'
```

1. Finally update anynines configuration package

```bash
kubectl patch configurations/w5n9a2g2-anynines-dataservices \
  --type merge -p '{"spec":{"package":"public.ecr.aws/w5n9a2g2/anynines/dataservices:v1.3.0"}}'
```

### Provider backend

:::warning

Please read the change log before updating, and follow any migration instructions there.

:::

1. Install the latest CRDs for the backend according to the
   [backend installation instructions](../central-management-cluster-setup/index.md#prerequisites-2)
2. Update the Klutch backend deployment according to the
   [installation instructions](../central-management-cluster-setup/index.md#deploy-the-klutch-backend)
3. If the new version also introduces new data service types, follow the binding creation steps
   [follow the binding creation steps](../central-management-cluster-setup/klutch-po-setup-tenant-cluster)
   to install them in consumer clusters.

## Downtime during update

:::note

While an update is in progress, the Crossplane providers and klutch-bind backend may incur a short
downtime. Note that this _does not_ affect any of the existing data service instances: they will
continue to be running and reachable. Any changes to the data service instances (creation, update,
deletion) that were made while an update is in progress will be applied as soon as the update is
complete.

:::
