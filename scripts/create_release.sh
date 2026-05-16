#! /usr/bin/env bash
set -euo pipefail

log-normal() {
    echo "[$(date "+%H:%M:%S")]" "$@"
}

check-dependency() {
    if ! command -v "$@" 1>/dev/null 2>&1; then
        log-normal "$* not found in \$PATH"
        exit 1
    fi
}

########
# INIT #
########

if [[ $# -lt 1 || $# -gt 2 ]]; then
    log-normal "Usage: create_release.sh <version number> [<AWS profile name]"
    exit 1
fi

VERSION_NUMBER="$1"
KLUTCHIO_REPO="$(git rev-parse --show-toplevel)"
DOCS_REPO="$KLUTCHIO_REPO/../klutchio-website"
ECR_REGISTRY_ADDRESS="public.ecr.aws/w5n9a2g2"
VERSION_CHECK_EXPRESSION='^v[0-9]\.[0-9]\.[0-9](-[-.A-z0-9]+)?$'

if ! echo "$VERSION_NUMBER" | grep -E "$VERSION_CHECK_EXPRESSION" -q; then
    log-normal "Illegal version number $VERSION_NUMBER. Please use a version number that matches this regular expression: $VERSION_CHECK_EXPRESSION"
    exit 1
fi

if [[ ! -d $DOCS_REPO ]]; then
    log-normal "Please make sure that the klutchio-website repo is cloned into the parent directory of the klutchio repo"
    exit 1
fi

if [[ $# -lt 2 ]]; then
    if [[ -z ${AWS_PROFILE:-} ]]; then
        log-normal "Detecting AWS account name..."
        AWS_PROFILE="$(aws configure list | grep "profile" | cut -d ':' -f 2 | awk '{$1=$1};1')"
        export AWS_PROFILE
    fi

    log-normal "No AWS profile specified, using active AWS profile $AWS_PROFILE"
else
    export AWS_PROFILE="$2"
fi

log-normal "Using AWS profile $AWS_PROFILE for interacting with the ECR registry $ECR_REGISTRY_ADDRESS"
check-dependency "ko"
check-dependency "crossplane"

#######################
# BUILD DOCKER IMAGES #
#######################

# log-normal "Logging into AWS ECR..."
# aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin ${ECR_REGISTRY_ADDRESS}
# log-normal "ECR login successful"

# cd "$KLUTCHIO_REPO/bind"

# log-normal "Building konnector image..."
# KO_DOCKER_REPO="${ECR_REGISTRY_ADDRESS}/klutch/konnector" ko build ./cmd/konnector --bare -t "${VERSION_NUMBER}" --platform=linux/amd64,linux/arm64
# log-normal "konnector image built successfully"

# log-normal "Building example-backend image..."
# KO_DOCKER_REPO="${ECR_REGISTRY_ADDRESS}/klutch/example-backend" ko build ./cmd/example-backend --bare -t "${VERSION_NUMBER}" --platform=linux/amd64,linux/arm64
# log-normal "example-backend image built successfully"

# make -C "$KLUTCHIO_REPO/crossplane-api" dataservices-config-push dataservicesConfigVersion="${VERSION_NUMBER}"

# cd "$KLUTCHIO_REPO/provider-anynines"
# log-normal "Building provider-anynines controller images..."
# make buildx.all IMAGETAG="${VERSION_NUMBER}"
# log-normal "provider-anynines controller images built successfully"
# log-normal "Building provider-anynines Provider image..."
# make provider-build-push IMAGETAG="${VERSION_NUMBER}"
# log-normal "provider-anynines Provider image built successfully"

####################
# UPDATE MANIFESTS #
####################

log-normal "Updating version references in klutchio manifests and README.md..."
yq -i ".spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/dataservices:$VERSION_NUMBER\"" \
    "$KLUTCHIO_REPO/crossplane-api/deploy/config-pkg-anynines.yaml"
yq -i "with(select(document_index == 0); .spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/provider-anynines:$VERSION_NUMBER\")" \
    "$KLUTCHIO_REPO/crossplane-api/deploy/provider-anynines.yaml"

README_PATH="$KLUTCHIO_REPO/crossplane-api/README.md"
sed -E 's|klutch/dataservices:v[0-9]+\.[0-9]+\.[0-9]+[^"]*|klutch/dataservices:'"${VERSION_NUMBER}"'|g' \
    "$README_PATH" > "${README_PATH}.tmp"
mv "${README_PATH}.tmp" "$README_PATH"

yq -i ".spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/provider-anynines:$VERSION_NUMBER\"" \
    "$KLUTCHIO_REPO/provider-anynines/examples/provider/provider.yaml"
yq -i ".spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/provider-anynines:$VERSION_NUMBER\"" \
    "$KLUTCHIO_REPO/test/e2e/provider/manifests/install/provider.yaml"
yq -i ".spec.package = \"$ECR_REGISTRY_ADDRESS/klutch/dataservices:$VERSION_NUMBER\"" \
    "$KLUTCHIO_REPO/test/e2e/provider/manifests/configuration.yaml"
log-normal "Klutchio manifest and README.md version references updated to ${VERSION_NUMBER}.\n"

########################
# UPDATE DOCUMENTATION #
########################

log-normal "Updating version references in klutchio-website documentation repo"
sed -E 's|klutch/konnector:v[0-9]+\.[0-9]+\.[0-9]+[^ ]*|klutch/konnector:'"${VERSION_NUMBER}"'|g' \
    "$DOCS_REPO/docs/local-deployment-guide.md" > "$DOCS_REPO/docs/local-deployment-guide.md.tmp"
mv "$DOCS_REPO/docs/local-deployment-guide.md.tmp" "$DOCS_REPO/docs/local-deployment-guide.md"

sed -E 's|klutch/provider-anynines:v[0-9]+\.[0-9]+\.[0-9]+[^ ]*|klutch/provider-anynines:'"${VERSION_NUMBER}"'|g' \
    "$DOCS_REPO/docs/platform-operator-guide/monitor-klutch-components.md" > \
    "$DOCS_REPO/docs/platform-operator-guide/monitor-klutch-components.md.tmp"
mv "$DOCS_REPO/docs/platform-operator-guide/monitor-klutch-components.md.tmp" \
    "$DOCS_REPO/docs/platform-operator-guide/monitor-klutch-components.md"

sed -E 's|https://anynines-artifacts.s3.eu-central-1.amazonaws.com/central-management/v[0-9]+\.[0-9]+\.[0-9]+[^/]*|https://anynines-artifacts.s3.eu-central-1.amazonaws.com/central-management/'"${VERSION_NUMBER}"'|g' \
    "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md" > \
    "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md.tmp"
mv "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md.tmp" \
    "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md"

sed -E 's|klutch/konnector:v[0-9]+\.[0-9]+\.[0-9]+[^ ]*|klutch/konnector:'"${VERSION_NUMBER}"'|g' \
    "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md" > \
    "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md.tmp"
mv "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md.tmp" \
    "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/app-cluster.md"

sed -E 's|klutch/example-backend:v[0-9]+\.[0-9]+\.[0-9]+[^ ]*|klutch/example-backend:'"${VERSION_NUMBER}"'|g' \
    "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/control-plane-cluster/index.md" > \
    "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/control-plane-cluster/index.md.tmp"
mv "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/control-plane-cluster/index.md.tmp" \
    "$DOCS_REPO/docs/platform-operator-guide/setting-up-klutch-clusters/control-plane-cluster/index.md"

####################
# UPDATE CHANGELOG #
####################

sed -E 's|^## Unreleased$|## '"[${VERSION_NUMBER}] - $(date "+%Y-%m-%d")|g" \
    "$KLUTCHIO_REPO/CHANGELOG.md" > "$KLUTCHIO_REPO/CHANGELOG.md.tmp"
mv "$KLUTCHIO_REPO/CHANGELOG.md.tmp" "$KLUTCHIO_REPO/CHANGELOG.md"

git add ./*
git commit -m "Prepare version $VERSION_NUMBER"
echo "# CHANGELOG

## Unreleased
$(tail -n +2 "$KLUTCHIO_REPO/CHANGELOG.md")" > "$KLUTCHIO_REPO/CHANGELOG.md"
