# provider-anynines

`provider-anynines` is a minimal [Crossplane](https://crossplane.io/) Provider that provides
functionality to create, update and delete VM based data service instances using Kubernetes
resources.

## Prerequisites

Install the following local binaries before running any build target:

- Git
- Go
- Docker
- Make
- [Crossplane CLI](https://docs.crossplane.io/latest/cli/)

For push workflows to AWS ECR Public, also install and configure:

- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)

## Mandatory First Step: Initialize Submodules

Run this before any `make` target in this directory:

```shell
git submodule update --init --recursive
```

This repository follows the common Crossplane/Upbound build-system pattern where shared Makefile
logic lives in the `build/` submodule. Without it, provider build targets cannot load required
make library files and variables.

## Build And Push Workflow

### 1) Build provider artifacts locally

```shell
cd provider-anynines
git submodule update --init --recursive
make build.all
```

### 2) Authenticate Docker to ECR Public

If needed, configure AWS credentials once:

```shell
aws configure --profile ECR
```

If you use a named profile:

```shell
export AWS_PROFILE=ECR
```

Login Docker to ECR Public:

```shell
aws ecr-public get-login-password --region us-east-1 --profile=ECR | docker login --username AWS --password-stdin public.ecr.aws/w5n9a2g2
```

### 3) Push controller image (multi-arch manifest)

Set a unique image tag:

```shell
export IMAGETAG=<your-initials-version>
# example: export IMAGETAG=AH-v0.0.1
```

Push controller images and manifest:

```shell
make provider-controller-push IMAGETAG=$IMAGETAG
```

### 4) Build and push provider package

```shell
make provider-build-push IMAGETAG=$IMAGETAG
```

## Required Variables For Push Workflows

- `IMAGETAG` (required): used by `provider-controller-push` and `provider-build-push`.
- `AWS_PROFILE` (optional): used for AWS CLI profile selection during registry login.

Registry authentication is required before running push targets.

## Troubleshooting

If `make` fails before target execution with errors like:

- missing make variables such as `INFO`
- `No rule to make target 'build/makefiles/...` (or similar `build/makelib/...` path errors)

then the build submodule is not initialized.

Fix:

```shell
git submodule update --init --recursive
```

Then rerun the make target.

## Developing

For day-to-day development tasks beyond the core build and push loop, see [Developing](./DEVELOPING.md).
