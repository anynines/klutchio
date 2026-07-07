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

# The PR must contain the version number and run-id as inputs for the Create Release workflow,
# which uses them for tagging the GH Release it creates and for retrieving the kubectl-bind-plugin
# binaries from the artifacts of the run that created the release PR.
# By enclosing these values in '[' and ']::' they will not be rendered when viewing the PR in a web
# browser but they will still be accessible to the Create Release workflow.
BODY="$(printf '[%s: %s]::\n' \
    "run-id" "${RUN_ID}" \
    "version-number" "${VERSION_NUMBER}")"

BODY+="This PR was created as part of the automated release process for version ${VERSION_NUMBER}."

gh pr create \
    --title "Release ${VERSION_NUMBER}" \
    --body "$BODY" \
    --base "$BASE_BRANCH" \
    --head "$TARGET_BRANCH" \
    --label "release"
