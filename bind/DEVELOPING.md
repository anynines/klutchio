# Developing Klutch Bind

## Make commands

`make codegen` has to be run after changes to the APIs for either klutch-bind or the example backend.

`make lint` to run the linter to make sure the code is up to standards. Each commit should pass this check.


## Working with the components in a dev environment

binaries are built using `make`
### Example Backend

Testing the backend requires a working and reachable OIDC provider. You can test the backend locally
by running it outside of the cluster. The CRDs `./deploy/crd` need to be installed in the cluster.
In order to do so run the backend locally you need to execute

```sh
go run ./cmd/example-backend \
    --oidc-issuer-client-secret "<oidc client secret>" \
    --oidc-issuer-client-id="<oidc client id>" \
    --oidc-issuer-url="<oidc issuer url>" \
    --oidc-callback-url="http://localhost:8080/callback" \
    --cookie-signing-key=$(openssl rand -base64 32) /

```
### Konnector

Most of the Konnector code is located in `./pkg/konnector`. To test out your own changes to the
Konnector build a new image using

```sh
KO_DOCKER_REPO=<your container registry> ko build ./cmd/konnector --bare -t <some new tag here>
```

Once you have a new image, you can either `kubectl edit -n bind deployment konnector` on your
consumer cluster and edit the image to your new image, or if you already have a consumer cluster
configured, or you can run `kubectl bind --konnector-image=<your image> <your backend>`

### Kubectl plugin

The kubectl plugin is used to bind a new cluster to a service provider. Most of the code is located
in `./pkg/kubectl`. Each subcommand has it's own folder. The entrypoint is at `./cmd/kubectl-bind`.
Kubectl automatically recognizes plugins by checking binaries in the path. It automatically
translates them into subcommands. E.g. `kubctl-bind` gets automatically translated into `kubectl
bind`

To try local changes, you can either:

run `make` to build a new version of the binary, and add `./bin` to your `PATH`, making sure it has
priority over any other `kubectl-bind` binaries that may be installed.

Or run the binary locally without using `kubectl` by using `go run ./cmd/kubectl-bind <args>`

## Automated testing

bind has unit tests and end to end tests. They can be executed using `make test` and `make test-e2e`
respectively. They should pass for each commit.
