#! /usr/bin/env bash
set -euo pipefail
# set -x

#####################
#                   #
# DECLARE CONSTANTS #
#                   #
#####################

DOCS_REPO="$(git rev-parse --show-toplevel)"
readonly DOCS_REPO
readonly DOCS_DIR="$DOCS_REPO/docs"
readonly SEMVER_EXP='v[0-9]\.[0-9]\.[0-9](-[-.A-z0-9]+)?'

#####################
#                   #
# DECLARE FUNCTIONS #
#                   #
#####################

log_info() {
    echo "[$(date "+%H:%M:%S")]" "$@"
}

log_fatal_error() {
    echo -e "[$(date "+%H:%M:%S")]"'\033[31m ERROR\033[0m:' "$@"
    exit 1
}

################################
#                              #
# SCRIPT EXECUTION STARTS HERE #
#                              #
################################

log_info "Updating version references in klutchio-website docs repo"
KON_REPO="klutch/konnector"
sed -E 's|'"$KON_REPO:$SEMVER_EXP"'|'"$KON_REPO:${VERSION_NR}"'|g' \
    "$DOCS_DIR/local-deployment-guide.md" \
    >"$DOCS_DIR/local-deployment-guide.md.tmp"
mv "$DOCS_DIR/local-deployment-guide.md.tmp" \
    "$DOCS_DIR/local-deployment-guide.md"

PROV_REPO="klutch/provider-anynines"
sed -E 's|'"$PROV_REPO:$SEMVER_EXP"'|'"$PROV_REPO:${VERSION_NR}"'|g' \
    "$DOCS_DIR/platform-operator-guide/monitor-klutch-components.md" > \
    "$DOCS_DIR/platform-operator-guide/monitor-klutch-components.md.tmp"
mv "$DOCS_DIR/platform-operator-guide/monitor-klutch-components.md.tmp" \
    "$DOCS_DIR/platform-operator-guide/monitor-klutch-components.md"

BUCKET_URL="https://anynines-artifacts.s3.eu-central-1.amazonaws.com"
BUCKET_DIR="$BUCKET_URL/central-management"
SETUP_DOCS="$DOCS_DIR/platform-operator-guide/setting-up-klutch-clusters"
sed -E \
    's|'"$BUCKET_DIR/$SEMVER_EXP"'[^/]*|'"$BUCKET_DIR/${VERSION_NR}"'|g' \
    "$SETUP_DOCS/app-cluster.md" >"$SETUP_DOCS/app-cluster.md.tmp"
mv "$SETUP_DOCS/app-cluster.md.tmp" "$SETUP_DOCS/app-cluster.md"

sed -E 's|'"$KON_REPO:$SEMVER_EXP"'[^ ]*|'"$KON_REPO:${VERSION_NR}"'|g' \
    "$SETUP_DOCS/app-cluster.md" >"$SETUP_DOCS/app-cluster.md.tmp"
mv "$SETUP_DOCS/app-cluster.md.tmp" "$SETUP_DOCS/app-cluster.md"

BACKEND_REPO="klutch/example-backend"
sed -E 's|'"$BACKEND_REPO:$SEMVER_EXP"'|'"$BACKEND_REPO:${VERSION_NR}"'|g' \
    "$SETUP_DOCS/control-plane-cluster/index.md" > \
    "$SETUP_DOCS/control-plane-cluster/index.md.tmp"
mv "$SETUP_DOCS/control-plane-cluster/index.md.tmp" \
    "$SETUP_DOCS/control-plane-cluster/index.md"

MSG="Update documentation references to version $VERSION_NR"
