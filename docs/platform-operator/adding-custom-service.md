---
id: add-custom-service
title: Adding a Custom Service
---

# Adding a Custom Service

You can also add your own services to Klutch. This can be useful in case you want to add an internal
service and distribute it the same way as the officially supported services. 

Services are offered in the form of Kubernetes Custom Resources. We recommend that you use
crossplane XRDs to define your service's interface for end users. This allows you to add services
already supported by crossplane, like public clouds, Kubernetes Operators, using
Provider-Kubernetes, or custom services by writing a crossplane provider wrapping an arbitrary API.
You can find a examples of this at `crossplane-api/api/{common,a8s}`. We recommend that you use
namespace scoped custom resources, to ensure that tenants can be properly isolated.

Once you have set up your custom API on the control plane cluster, you need to make it available for
binding for your users. In order to do that you can create an `APIServiceExportTemplate` custom
resource on the control plane cluster. This lets klutch-bind know that you want to share the API
with users. To make an API available for sharing to users, you can create a resource like this:

```yaml
apiVersion: example-backend.klutch.anynines.com/v1alpha1
kind: APIServiceExportTemplate
metadata:
  name: <choose a descriptive name>
  namespace: crossplane-system
spec:
  APIServiceSelector:
    group: <your api group>
    resource: <your resource name(plural)
    version: <your resource version>
```

Applying this custom resource to the control plane cluster will make your API available for binding
using the web interface. In this base configuration only the resources of that type get synchronized
to the app cluster. If your API requires additional resources to be synchronized, for example a
secret with connection details you need to configure the synchronization for that resource. To add
additional resource for synchronization, you can add a "permission claim" to your
`APIServiceExportTemplate` to let klutch-bind claim the permission to sync another resource. The
example below shows the "servicebindings" API shared via klutch-bind, with the additional permission
claims to synchronize secrets and config maps from the control plane cluster to the app cluster.
Syncing of claimed resources always includes all resources of that type in all bound namespaces.

```yaml
kind: APIServiceExportTemplate
apiVersion: example-backend.klutch.anynines.com/v1alpha1
metadata:
  name: "servicebindings"
  namespace: crossplane-system
spec:
  APIServiceSelector:
    resource: servicebindings
    group: anynines.com
  permissionClaims:
    - group: ""
      resource: secrets
      version: v1
      selector:
        owner: Provider
    - group: ""
      resource: configmaps
      version: v1
      selector:
        owner: Provider
```
