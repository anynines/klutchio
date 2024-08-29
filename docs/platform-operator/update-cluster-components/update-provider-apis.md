---
id: klutch-po-update-provider-apis
title: "Background Info: API binding propagation"
tags:
  - provider cluster
  - kubernetes
  - a9s data services
  - platform operator
keywords:
  - a9s data services
  - platform operator
---

:::note

This page contains background information that may be useful to gain a better understanding of the
system or for debugging. This information is not needed for normal system operation.

:::

## Journey of an API

In this section we will cover the journey of an API definition through the stack. Initially, the API
is defined in a Crossplane Configuration Package. When this package is installed to the central
management cluster, the API definitions are extracted by Crossplane. To make the resource packages
available for tenant clusters, the Platform Operator defines an `APIServiceExporttemplate`. When a
binding is created the tenant cluster will create an `APIServiceExportRequest` on the central
management cluster.

Upon creation of the `APIServiceExportRequest` the Klutch backend will grant the tenant cluster's
Kubernetes service account the necessary permissions to interact with the requested API and its
related objects. Afterwards the Klutch backend creates an `APIServiceExport` object that contains a
snapshot of the bound CRD at the time of binding.

The application developer then applies an `APIServiceBinding` object to their cluster. In the
binding process this is done by executing the `kubectl bind` command. This event is picked up by the
`Konnector` installed in the tenant cluster. The `Konnector` will read the `APIServiceBinding`
object and attempt to find a matching `APIServiceExport` on the central management cluster. If a
matching Object is found the `Konnector` reads the API schema from the `APIServiceExport` and
creates a Custom Resource Definition (CRD) with a matching schema on the tenant cluster. This
process runs continuously and will pick up changes and new APIs as they are added.

## Updating provider APIs

Updating of API major versions is currently not supported. Updates containing non-breaking changes
to the API are supported.

:::warning

Breaking API changes are not prevented automatically. Please make sure that the API you are updating
does not have any breaking changes.

:::

Klutch will not introduce breaking changes to the data services APIs until safe migrations are
supported.

Coming soon, updates to APIs on the central management cluster will be automatically distributed to
the tenant clusters.

When a change to a CRD that is referenced by an `APIServiceExportTemplate` is detected, all
`APIServiceExport`s will be modified to include the new change. The `Konnector` on tenant clusters
will detect this change in the `APIServiceExport` and update the local CRDs acccordingly.

## Adding new APIs

Adding a new API - e.g. a new data dervice - requires a new binding creation. This means the
creation of a `APIServiceExport` on the central management cluster and the creation of a
`APIServiceBinding` on the tenant cluster.

The easiest way to create them is to follow the process for new bindings using `kubectl bind` as
described above.