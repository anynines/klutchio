# provider-anynines

`provider-anynines` is a minimal [Crossplane](https://crossplane.io/) Provider
that provides functionality to create, update and delete VM based data service
instances using Kubernetes resources.

## Developing

This section explains the basic commands needed while developing the provider,
the [Provider Development][provider-dev] guide may also be of use.

### Test & Build

- Run `make generate` to generate/re-generate CRDs.
- Run `make reviewable` in `/provider-anynines` to run code generation, linters, and tests.
- Run `make build` in `/provider-anynines` to build the provider.

Note: *If you face problems executing the commands mentioned above you can temporarily remove
the `go.work` file from the root directory and execute the commands again.*

### Adding new API types

To add a new type to the API by run the following command:

```
make provider.addtype provider={provider-anynines} group={group} kind={type}
```

Note: *If you face problems executing the above mentioned cmd you can temporarily remove
the `go.work` file from the root directory and execute the cmd again.*

## Files

There are a number of build tools and processes that are common across the
Crossplane ecosystem. Using these ensures a consistent development environment
across projects.

This repository follows the [provider-template](https://github.com/crossplane/provider-template)
and therefore contains:

- The [Upbound build](https://github.com/upbound/build) submodule. (see
  [https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#establishing-a-development-environment](https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#establishing-a-development-environment))
- A [Makefile](https://github.com/crossplane/provider-gcp/blob/master/Makefile)
  that supports common build targets.
- A Golang linter. Example:
  [https://github.com/crossplane/provider-aws/blob/master/.golangci.yml](https://github.com/crossplane/provider-aws/blob/master/.golangci.yml)
- A [Crossplane Package](https://crossplane.io/docs/master/concepts/packages.html)
  configuration (see
  [package/crossplane.yaml)](https://github.com/crossplane/provider-template/blob/main/package/crossplane.yaml)
- Examples for the ProviderConfig and each resource in the
  `examples/` directory.

## Limitations

### UID collisions

The a9s provider uses the Kubernetes generated UIDs as UIDs for data service
instances and Service Bindings. As UIDs are generally speaking unique (see
[Kubernetes
Documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/)) 
in space and in time a collision should not occur. However, as the a9s Service
Broker stores all UIDs for auditing purposes, at some point a collision with
deleted instances might occur. In such a case, the instance will be marked as 
unavailable and will never become available. This can be solved by simply 
deleting and recreating the corresponding object.

The chance of such a collision however is extremely low. More information can be
found in:

- https://github.com/kubernetes/design-proposals-archive/blob/main/architecture/identifiers.md
- https://www.ietf.org/rfc/rfc4122.txt


[provider-dev]: https://docs.crossplane.io/v1.10/contributing/provider_development_guide/
