#! /usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 <version-number> <base-branch>"
    exit 1
fi

VERSION_NUMBER=$1
BASE_BRANCH=$2
TARGET_BRANCH="releases/${VERSION_NUMBER}"

gh pr create \
    --title "Release ${VERSION_NUMBER}" \
    --body "This PR was created as part of the automated release process for version ${VERSION_NUMBER}." \
    --base "$BASE_BRANCH" \
    --head "$TARGET_BRANCH"
