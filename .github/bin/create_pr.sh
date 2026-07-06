#! /usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
    echo "Usage: $0 <version-number> <base-branch> <run-id>"
    exit 1
fi

VERSION_NUMBER=$1
BASE_BRANCH=$2
RUN_ID=$3
TARGET_BRANCH="releases/${VERSION_NUMBER}"

BODY="$(printf '[run-id: %s]::
[version-number: %s]::
This PR was created as part of the automated release process for version %s.\n' \
    "${RUN_ID}" "${VERSION_NUMBER}" "${VERSION_NUMBER}")"

gh pr create \
    --title "Release ${VERSION_NUMBER}" \
    --body "$BODY" \
    --base "$BASE_BRANCH" \
    --head "$TARGET_BRANCH" \
    --label "release"
