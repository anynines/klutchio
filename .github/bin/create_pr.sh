#! /usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
    echo "Usage: $0 <version-number>"
    exit 1
fi

VERSION_NUMBER=$1
BRANCH_NAME="releases/${VERSION_NUMBER}"

gh pr create \
    --title "Release ${VERSION_NUMBER}" \
    --body "This PR was created as part of the automated release process for version ${VERSION_NUMBER}." \
    --base main-test \
    --head "$BRANCH_NAME"
