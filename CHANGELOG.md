# CHANGELOG

## Unreleased

## [0.0.2-jlu-1780670831] - 2026-06-05

### Changed

- **breaking**: `providerconfigs.dataservices.anynines.com` now expects a field `spec.serviceType`, which can be either `servicebroker` or `backupmanager`.
- Added TLS support for communication with service-broker in provider-anynines. Service-broker URL can now use https.
- Klutch-bind: advanced konnector control plane mode with explicit client separation for control plane, binding cluster, and app cluster paths, plus fixes for APIServiceBinding writes in both modes.
- Klutch-bind: migrated APIServiceBinding handling to namespace scope as part of control plane mode hardening.
- Klutch-bind: updated control plane mode root namespace handling for app cluster kubeconfig and simplified AppClusterBinding RBAC.

## [1.5.0] - 2026-05-26

### Added

- Added TLS support for communication with service-broker in provider-anynines. Service-broker URL can now use https.
- Added Custom Parameter Support for a9s KeyValue
- **breaking**: Added required field `spec.serviceType` to
  `providerconfigs.dataservices.anynines.com`, which can either have the value `servicebroker` or
  `backupmanager`.

### Changed

- Bumped a9s Data Service integration to 1.4.1: update supported services: add a9s-messaging 4
- Improved error handling in UID check
- Made `spec.apiServiceSelector.version` optional in example-backend ApiServiceExportTemplate CRD
- switched out the retired [ingress-nginx](https://github.com/kubernetes/ingress-nginx/) for [Envoy
  Gateway](https://github.com/envoyproxy/gateway) in the `bind` example backend manifests

### Fixed

- Fixed `bind` e2e tests
- Fixed typo in example Provider manifest for `provider-anynines`
- **breaking** Fixed typo in a9s postgresql composition
- Updated and fixed `bind` code generation

### Update Dependencies

- Bumped github.com/moby/spdystream in `bind` to close [CVE-2026-35469](https://nvd.nist.gov/vuln/detail/CVE-2026-35469)
- Bumped go.opentelemetry.io/otel/sdk in `bind` to close [CVE-2026-24051](https://nvd.nist.gov/vuln/detail/CVE-2026-24051) and [CVE-2026-39883](https://nvd.nist.gov/vuln/detail/CVE-2026-39883)
- Bumped github.com/go-jose/go-jose/v4 in `bind` to close
[CVE-2026-34986](https://nvd.nist.gov/vuln/detail/CVE-2026-34986)
- Bumped google.golang.org/grpc in `bind` and `provider-anynines` to close [CVE-2026-33186](https://nvd.nist.gov/vuln/detail/CVE-2026-33186)

### Chores

- Bump go version of components to v1.26.3.

## [1.4.0] - 2025-12-18

### Changed

- **deprecated** `status.forProvider.serviceBindingID` on the `ServiceBinding` resource. Use `status.forProvider.serviceID` instead.
- **breaking**: ServiceBinding resources now use semantic connection labels (e.g., "SQL", "Logme2", "Search") for improved clarity in status and connection secrets.
- Updated `ServiceInstance` reconciliation logic to use the latest endpoints of the `ServiceBroker`.
- All connection details in ServiceBinding status and secrets are now consistently labeled for easier integration and troubleshooting.
- Provider-anynines: Make health check endpoint configurable in the ProviderConfig spec.
- Provider-anynines: Increase default request timeout.

### Compatibility

- This version is compatible **only with a9s Data Services [v68.0.0](https://docs.anynines.com/changelog) and later** (only valid for a9s DataServices integration).
- Older versions of the provider remain compatible with a9s Data Services v68.x.

### Chores

- Upgraded Crossplane to version [v1.20.0](https://docs.crossplane.io/v1.20).
- Updated crossplane functions
  - `function-patch-and-transform` upgraded to v0.9.2
- Fixed security vulnerabilities identified by Dependabot.

### Added

- Added TLS support for communication with backup-manager in provider-anynines. Changing the URL in
  the ProviderConfig from http:// to https:// switches to encrypted communication. A new optional
  configuration field `tls` has been added to configure custom certificates.
- Added `KeyValue` integration to a9s DataServices, enabling seamless key-value data management.
- Added `Custom Parameters` to `LogMe2`, `Search`, `Prometheus`, `MariaDB`, `Messaging`, and `MongoDB`,
 allowing users to tailor integration settings.

## [1.3.2] - 2025-01-23

### Added

- Added `HostURL` and `Port` fields to the `ServiceBinding` status of `provider-anynines` managed resources.

### Changed

- Fix composition resources in './crossplane-api' to use new mapping paths.

## [1.3.1] - 2025-01-14

### Added

- Added documentation for dynamic Kubernetes clients, providing guidance on usage and best practices.

### Changed

- Aligned PostgreSQL plan names with a9s-dataservices for improved consistency and integration.

- Adjusted health check timeouts for provider-anynines to improve reliability and reduce errors.

- Enhanced internal processes by updating the Makefile for optimized pipeline management.

- Replaced native patch and transform functionality with composite functions from Crossplane, ensuring better modularity and maintainability.

## [1.3.0] - 2025-01-14

### Changed

- Renamed image repositories prefix from `public.ecr.aws/w5n9a2g2/anynines` to `public.ecr.aws/w5n9a2g2/klutch`.

- Removed unused resources.

- Updated naming conventions. `Consumer Cluster` is now `App Cluster` and `Management Cluster` is now `Control Plane Cluster`.

- Renamed backend resources for bindings.
  Changed the namespace used for bindings on the App Clusters from `kube-bind` to `klutch-bind`.

  This change automatically applies to new bindings.

  **breaking** this change also changes the namespace of the `konnector` deployment. Please make
  sure that only one deployment of `konnector` is running. Please delete the old konnector
  deployment by running

  ```sh
  kubectl delete -n kube-bind deployment konnector
  ```

  before creating any new bindings. Creating a new binding will deploy the konnector to the new
  namespace.

## [1.0.0] - 2024-08-30

### Added

- Crossplane Provider named `provider-anynines` for leveraging a9s Data Services.
- Health checks and readinessProbe for ProviderConfigs.
- Klutch-bind for cross-cluster service management.
- Crossplane APIs for a9s Data services (provider-anynines), a8s Data Services (provider-kubernetes) and AWS s3 buckets (provider-aws-s3).
- Documentation content for the Docusaurus-powered Klutchio website.
- End-to-end tests for Klutch.
