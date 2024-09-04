# Developing

This page explains the basic commands needed while developing the provider, the [Provider
Development][provider-dev] guide may also be of use.

## Requirements

- the [Makefile](./Makefile) for `provider-anynines` requires
  [Docker Desktop](https://www.docker.com/products/docker-desktop/) to be installed on the local
  system.
- the [Makefile](./Makefile) for `provider-anynines` requires
  [Crossplane CLI](https://docs.crossplane.io/latest/cli/) to be installed on the local system.

## Initialization

Execute `make submodules` to initialize
["the build submodule" i.e. Crossplane's Makefile library](https://github.com/crossplane/crossplane-build):

```shell
$ make submodules
Submodule 'provider-anynines/build' (https://github.com/crossplane/build) registered for path 'build'
Cloning into '<path to>/klutch/provider-anynines/build'...
```

## Test & Build

- Run `make generate` to generate/re-generate CRDs.
- Run `make reviewable` in `/provider-anynines` to run code generation, linters, and tests.
- Run `make build` in `/provider-anynines` to build the provider.

Note: _If you face problems executing the commands mentioned above you can temporarily remove the
`go.work` file from the root directory and execute the commands again._

## Adding new API types

To add a new type to the API by run the following command:

```
make provider.addtype provider={provider-anynines} group={group} kind={type}
```

Note: _If you face problems executing the above mentioned cmd you can temporarily remove the
`go.work` file from the root directory and execute the cmd again._

## Files

There are a number of build tools and processes that are common across the Crossplane ecosystem.
Using these ensures a consistent development environment across projects.

This repository follows the [provider-template](https://github.com/crossplane/provider-template) and
therefore contains:

- The [Crossplane build](https://github.com/crossplane/build) submodule. (see
  [here for more information](https://github.com/crossplane/crossplane/tree/master/contributing#establishing-a-development-environment))
- A [Makefile](https://github.com/crossplane/provider-gcp/blob/master/Makefile) that supports common
  build targets.
- A Golang linter. Example:
  [https://github.com/crossplane/provider-aws/blob/master/.golangci.yml](https://github.com/crossplane/provider-aws/blob/master/.golangci.yml)
- A [Crossplane Package](https://crossplane.io/docs/master/concepts/packages.html) configuration
  (see
  [package/crossplane.yaml](https://github.com/crossplane/provider-template/blob/main/package/crossplane.yaml))
- Examples for the ProviderConfig and each resource in the directory [examples](./examples).

## Limitations

### UID collisions

The a9s provider uses the Kubernetes generated UIDs as UIDs for data service instances and Service
Bindings. As UIDs are generally speaking unique (see
[Kubernetes Documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/))
in space and in time a collision should not occur. However, as the a9s Service Broker stores all
UIDs for auditing purposes, at some point a collision with deleted instances might occur. In such a
case, the instance will be marked as unavailable and will never become available. This can be solved
by simply deleting and recreating the corresponding object.

The chance of such a collision however is extremely low. More information can be found in:

- https://github.com/kubernetes/design-proposals-archive/blob/main/architecture/identifiers.md
- https://www.ietf.org/rfc/rfc4122.txt

[provider-dev]:
  https://github.com/crossplane/crossplane/blob/master/contributing/guide-provider-development.md
