# CHANGELOG

## Unreleased

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
- Klunch-bind for cross-cluster service management.
- Crossplane APIs for a9s Data services (provider-anynines), a8s Data Services (provider-kubernetes) and AWS s3 buckets (provider-aws-s3).
- Documentation content for the Docusaurus-powered Klutchio website.
- End-to-end tests for Klutch.
