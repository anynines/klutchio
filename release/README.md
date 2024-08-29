# Releases

## Terms

- "local variables": variables that are local to the environment where something is installed (i.e. they are not known when the release is built)
- "bundle": a collection of YAML files, concatenated into a single YAML file, with `---` markers

## Anatomy of a release

A release consists of multiple bundles:
- `install`: Contains everything that does not require any CRDs to be installed (except for crossplane, which is a prerequisite)
- `install-dev`: Everything included in `install`, plus additional things specific to a local development installation
- `configure`: Contains everything that either:
  1. requires a CRD to be installed, which is part of the `install` bundle
  2. contains local variables
- `update`: All parts from `install` that need to be changed during an update. For simplicity, contains everything that references an image (crossplane package,  container image, ...)

Releases are versioned using [semver](https://semver.org/).

### Patch releases

A patch release may contain only changes to `install` and `update`.

Any change to `configure` requires a minor release. This way we ensure that patch releases can be applied without any user intervention.

## Creating a release

1. Check if any of the `bases` need to be changed:
   - any new or updated CRDs?
   - any new config maps, secrets?
   - ...
2. Decide how to name the version:
   - Only fixes to existing functionality, and no changes to the `configure` bundle? -> PATCH release
   - Contains new features (backwards compatible), or fixes which require manual intervation? -> MINOR release
   - Contains backwards incompatible changes? -> MAJOR release
   - Exception: a MAJOR release can also be created for marketing or sales reasons. It doesn't *have* to contain breaking changes.
   - Before creating a "real" release, create a release candidate (append `-rc1`, `rc2`, ... to the release version).
3. Create images and packages for all components of the release
4. Export variables for all component versions (versions == tags here):
```
export CONFIG_VERSION=...
export PROVIDER_ANYNINES_VERSION=...
export PROVIDER_KUBERNETES_VERSION=...
export KUBE_BIND_BACKEND_VERSION=...
```
5. Create a release bundle `./release/scripts/make-release-bundle.sh <semver-version>`

You'll find the resulting files in `./release/output/`.

## Scenarios

### Controlled patch rollout

1. Suppose we are on version 1.2.3.
2. Customer X reports bug B
3. We fix bug B and publish version 1.2.4
4. We notify customer X, asking them to update explicitly to 1.2.4
5. The customer notifies us that the bug is fixed
6. We update the `1.2` version link to point to 1.2.4
7. The patch gets rolled out to all customers using automated updates
