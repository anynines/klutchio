# Validation tests

This directory contains tests for checking validation of the resources produced by our crossplane XRDs.
The XRDs under test can be found in `crossplane-api/api`.

## Prerequisites

### Host prerequites

The following tools are needed for the steps below:
- kind or minikube
- [Helm](https://helm.sh/docs/intro/install/), for installing the crossplane helm chart
- [yq](https://github.com/mikefarah/yq), for processing YAML files in shell scripts

### Cluster Prerequisites

Things that must be installed on the cluster:
- crossplane
- the anynines compositions & XRDs

It's not necessary to run any crossplane provider.

Copy & paste instructions for the above are:
```
helm repo add crossplane-stable https://charts.crossplane.io/stable && helm repo update
helm install crossplane --namespace crossplane-system --create-namespace crossplane-stable/crossplane --version 1.14.1
kubectl -n crossplane-system wait --for=condition=available deployment/crossplane deployment/crossplane-rbac-manager
kubectl apply --recursive -f ./crossplane-api/api/common
kubectl apply --recursive -f ./crossplane-api/api/a8s
kubectl apply --recursive -f ./crossplane-api/api/a9s
```

## Running the tests

To run all of the tests, use:
```
./test/validations/run-tests.sh
```

This invokes two other scripts: `run-create-tests.sh` and `run-update-tests.sh`

To run just one type of tests, invoke that script directly.

To check just a subset of the test cases, pass the respective test files to the correct `run-*-tests.sh` script.

Both scripts will print the output of their `kubectl` interactions to stderr if `TEST_VERBOSE=true` is set via the environment.

## What do the tests do?

There are two types of tests: "create" and "update" tests.

### "Create" tests

These tests check the creation of new objects.

Any file that matches the pattern `*/create-{valid,invalid}-*.yaml` is considered a "create" test.

The test runner will try to apply each file's contents (using `--dry-run`, so nothing is actually created).

If the filename starts with `create-valid-`, it must apply cleanly.
If it starts with `create-invalid-` it must produce an error.

### "Update" tests

These tests check updating an existing resource.

Any file that matches the pattern `*/update-*.yaml` is considered an "update" test.

Update tests follow this template:
```yaml
# `base` defines the object state before any updates. The manifest must be valid, as it will actually be applied
# to the test cluster.
base:
  apiVersion: ...
  kind: ...
  spec:
    plan: nano
    service: pg14

# `invalid_patches` and `valid_patches` are two lists of patches (the kind that `kubectl patch --type=merge` accepts).
# Each of them will be applied (with `--dry-run=server`). The ones listed under `invalid_patches` must fail,
# the ones from `valid_patches` must succeed.

invalid_patches:
- spec:
    service: pg15

valid_patches:
- spec:
    plan: small
```
