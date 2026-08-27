#! /usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <version_number> <ecr_registry_address>"
    exit 1
fi

VERSION_NR="$1"
ECR_REGISTRY_ADDRESS="$2"

PROVIDER_IMAGE="${ECR_REGISTRY_ADDRESS}/klutch/provider-anynines:${VERSION_NR}"

yq -i \
    "with(select(document_index == 0); .spec.package=\"$PROVIDER_IMAGE\")" \
    "crossplane-api/deploy/provider-anynines.yaml"

yq -i ".spec.package = \"$PROVIDER_IMAGE\"" \
    "provider-anynines/examples/provider/provider.yaml"
yq -i ".spec.package = \"$PROVIDER_IMAGE\"" \
    "test/e2e/provider/manifests/install/provider.yaml"
