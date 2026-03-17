# Release Process for Klutch

This document outlines the steps to release Klutch, which involves releasing its sub-components in a specific order.

1. **Release Bind Component:**
   Follow the release instructions in the `bind` directory:
   [bind/RELEASE.md](./bind/RELEASE.md)

2. **Sync a9s Services and Plans:**
   Ensure that the supported a9s Data Services and their plans, as defined in the Crossplane configuration packages, are aligned with the latest available a9s services and plans.
   Refer to the a9s Data Service lifecycle table for validation:  
   <https://docs.anynines.com/docs/platform-operator/a9s-po-sunrise-sunset/#a9s-data-service-release-lifecycle-table>

3. **Release Crossplane API Component:**
   Follow the release instructions in the `crossplane-api` directory:
   [crossplane-api/RELEASE.MD](./crossplane-api/RELEASE.md)

4. **Update Documentation:**
   Update the [`docs`](./docs) folder with the new image tags (versions) for all released components.
   **Important:** Ensure all providers and functions are also synced and use consistent versions across the release.

5. **Update Changelog:**
   - Review merged Pull Requests and ensure all changes are reflected in the [`CHANGELOG.md`](./CHANGELOG.md).
   - Replace `## [Unreleased]` with the actual version number (e.g., `## [v1.5.0] - 2026-01-26`).
   - After the release PR is merged and the release is published, add a new `## [Unreleased]` section at the top of the changelog.
