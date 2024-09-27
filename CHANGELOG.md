# CHANGELOG

## Unreleased

### Changed

- updated naming conventions. `Consumer Cluster` is now `App Cluster` and `Management Cluster` is now `Control Plane Cluster`.

- renamed backend resources for bindings.
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

## [1.3.0] - 2024-07-10

### Changed

- *breaking* change API groups for binding (formerly kube-bind)

## [1.2.0] - 2024-03-06

### Added

- Add support for PostgreSQL extensions via parameters field

### Chores

- Added support for Server-Side Apply (SSA) by upgrading Crossplane to version [v1.15.0](https://docs.crossplane.io/v1.15/release-notes/docs/)
- Updated all internal dependencies and packages to their latest compatible versions

## [1.1.0] - 2024-01-11

### Added

- Add health check for ProviderConfigs
- Add basic readinessProbe

### Changed

- Improve user-friendliness of a9s Messaging created secrets
- Extend validations for 'plan' and 'service' for all supported DSIs
- Add validations for postgresql claim

### Fixed

- Log sanitization for anynines-backend to exclude any recording of confidential information 

## [1.0.0] - 2023-10-20

### Added

- provider anynines: Add support for additional Data Services:
  - a9s Logme2
  - a9s MariaDB
  - a9s Messaging
  - a9s MongoDB
  - a9s Prometheus
  - a9s Search
- docs: Application Developer: all services: Add instructions for using the supported a9s Data Services through
Kubernetes. This includes setting up a Kubernetes environment, offering templates and examples for interacting with the
Data Service Instances and presenting tables with supported plans and services. For more information see Application
Developer.
- docs: Application Developer: all services: Add "Coming Soon" section for each Data Service, outlining upcoming
features. For more information see Application Developer.

### Changed

- all services: Update the Crossplane Configuration Package with the latest version of provider-anynines and enable
support for the additional a9s Data Services.
- all services: Update the API group and version for claims, compositions, and managed resources.
- all services: Improve readability of error messages.
- all services: Update provider-anynines container images to provide multi-architecture support.
- all services: Update build, push, and installation scripts to include the additional a9s Data Services.
- docs: Platform Operators: all services: Update the Platform Operators documentation, which now uses a single page to
deliver information on configuring Central Management and Tenant clusters with the supported a9s Data Services. For more
information see Setting up Central Management and Tenant Clusters.
- docs: Platform Operators: all services: Update the Platform Operators documentation by introducing a "Coming Soon"
section that outlines upcoming supported features. For more information see Setting up Central Management and Tenant
Clusters.

### Fixed

- all services: ServiceBinding: Fix ServiceBinding to use the same Kubernetes namespace as the Composition Claim. This
ensures that ServiceBindings work seamlessly in both Tenant and Central Management clusters synchronized with
kube-bind.

## [0.1.0] - 2023-08-23

### Added

- provider anynines: Add a Crossplane Provider named "provider-anynines" tailored to utilize a9s Data Services.
- provider anynines: Add build, push, and installation scripts, along with instructions for locally deploying
provider-anynines.
- a9s PostgreSQL: Add Crossplane Configuration Packages for the installation of provider-anynines and the necessary
Configuration Package for integrating a9s PostgreSQL with Kubernetes.
- a9s PostgreSQL: Add examples for provisioning a9s PostgreSQL Data Service Instances, creating service bindings,
performing backups and restoring data. For more information see Using a9s PostgreSQL.
- a9s PostgreSQL: Add a demo scenario that showcases the process of establishing a Central Management Cluster on Amazon
EKS, utilising Crossplane, Kube-bind, and the a8s framework.
- docs: Application Developer: a9s PostgreSQL: Add instructions for using a9s Data Services through Kubernetes. This
includes setting up a Kubernetes environment, provided templates and examples for interacting with the data service and
providing tables with supported plans and services. For more information see Using a9s PostgreSQL.
- docs: Application Developer: a9s PostgreSQL: Add a new "Coming Soon" section describing upcoming features. For more
information see Using a9s PostgreSQL.
- docs: Platform Operators: Add instructions for setting up a Central Management cluster and the Tenant cluster. For
more information see Setting up Central Management and Tenant Clusters.
