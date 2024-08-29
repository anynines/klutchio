# klutch e2e tests

This directory contains klutch e2e tests. It contains:

1. Yaml manifests for klutch
2. Tests to verify resulting resources work are deployed as expected.

## How to run the postgresql tests

The tests can be executed in a local `kind` cluster. We assume that these tests
can be deployed in an environment where a Kubernetes cluster already exists.

- Follow the instructions from [this repo](https://github.com/anynines/klutch/tree/main/crossplane-api) to setup all the prerequisites and the actual provider-anynines. You will need to follow the steps outlined in the [Prerequisites](https://github.com/anynines/klutch/tree/main/crossplane-api#prerequisites) and [Installation](https://github.com/anynines/klutch/tree/main/crossplane-api#installation) sections.

- You also need to set some environment variables:
  - KUBECONFIG

    ```bash
    export KUBECONFIG=${HOME}/.kube/config #Usually
    ```

- You can run the tests with the command:

  ```bash
  go test -v -timeout 99999s ./postgresql
  ```

## How to run the provider tests

The following tools need to be installed:
- `kind`: used to create a test cluster
- `helm`: used to install crossplane

To run the tests, execute:
```bash
go test -v ./provider
```

The test will create a fresh `kind` cluster and install crossplane together with the provider in there.
After the test completes successfully, the cluster is deleted.

In case of a failed test run the cluster is not deleted, allowing you to inspect it.

If you want to keep the cluster around even after a successful test run, set `KEEP_CLUSTER=true`:
```bash
KEEP_CLUSTER=true go test -v ./provider
```

To clean up all clusters left behind by e2e tests in one go, run:
```
make cleanup-clusters
```
