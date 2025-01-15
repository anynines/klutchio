# Dynamic Client

Klutch Bind uses client-go's `dynamic` client, a dynamic client is a kubernetes client that does not
have an associated type. Klutch-bind uses it in the Konnector to sync arbitrary resources between
clusters. The resources to be synced are configured at runtime by the APIs that the user has bound,
so they cannot be pre-compiled.

Instead of working on objects of a struct, it takes and returns `map[string]interface{}`. The string
key is the field name, and the interface is the value of the field. For example: `obj["spec"]` will
return the spec, which will be another `map[string]interface{}`. Because the client does not have an
associated type, it needs to be parameterized with the `GroupVersionKind` for operations.

Inside the Konnector, klutch-bind automatically configures and starts new controllers for each
resource to be synchronized based on dynamic client-go clients. To learn more about how controllers
are constructed using client-go you can check out the following resources:

- [kubernetes sample controller using
  client-go](https://github.com/kubernetes/sample-controller/blob/master/docs/controller-client-go.md)
- [client-go dynamic](https://github.com/kubernetes/client-go/tree/master/examples/dynamic-create-update-delete-deployment)
- [kubecon talk about client-go controllers](https://www.youtube.com/watch?v=_BuqPMlXfpE)
