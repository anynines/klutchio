#! /usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <version_number> <ecr_registry_address>"
    exit 1
fi

VERSION_NR="$1"
ECR_REGISTRY_ADDRESS="$2"
SEMVER_EXP='v[0-9]\.[0-9]\.[0-9](-[-.A-z0-9]+)?'

DS_BUNDLE_REPO="klutch/dataservices"
README_PATH="crossplane-api/README.md"
DS_BUNDLE_IMAGE="${ECR_REGISTRY_ADDRESS}/${DS_BUNDLE_REPO}:${VERSION_NR}"
PROVIDER_IMAGE="${ECR_REGISTRY_ADDRESS}/klutch/provider-anynines:${VERSION_NR}"

yq -i ".spec.package = \"$DS_BUNDLE_IMAGE\"" \
    "crossplane-api/deploy/config-pkg-anynines.yaml"
yq -i \
    "with(select(document_index == 0); .spec.package=\"$PROVIDER_IMAGE\")" \
    "crossplane-api/deploy/provider-anynines.yaml"

sed -E 's|'"$DS_BUNDLE_REPO:$SEMVER_EXP"'|'"$DS_BUNDLE_REPO:${VERSION_NR}"'|g' \
    "$README_PATH" >"${README_PATH}.tmp"
mv "${README_PATH}.tmp" "$README_PATH"

yq -i ".spec.package = \"$PROVIDER_IMAGE\"" \
    "provider-anynines/examples/provider/provider.yaml"
yq -i ".spec.package = \"$PROVIDER_IMAGE\"" \
    "test/e2e/provider/manifests/install/provider.yaml"
yq -i ".spec.package = \"$DS_BUNDLE_IMAGE\"" \
    "test/e2e/provider/manifests/configuration.yaml"
