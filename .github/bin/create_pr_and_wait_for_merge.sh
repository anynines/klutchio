#! /usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 <version-number> <approver-token>"
    exit 1
fi

VERSION_NUMBER=$1
APPROVER_TOKEN=$2

BRANCH_NAME="releases/${VERSION_NUMBER}"

OUTPUT="$(gh pr create \
    --title "Release ${VERSION_NUMBER}" \
    --body "This PR was created as part of the automated release process for version ${VERSION_NUMBER}." \
    --base main-test \
    --head "$BRANCH_NAME")"

echo "$OUTPUT"

PR_NUMBER="${OUTPUT##*/}"

# echo "Waiting for PR #$PR_NUMBER to be merged..."
# while true; do
#     MERGED_AT=$(gh pr view "$PR_NUMBER" --json mergedAt -q '.mergedAt')
#     if [[ "$MERGED_AT" != "null" && "$MERGED_AT" != "" ]]; then
#         echo "PR merged!"
#         break
#     fi
#     echo "Not merged yet. Waiting 10s..."
#     sleep 10
# done

export GH_TOKEN="$APPROVER_TOKEN"
gh pr review "$PR_NUMBER" --approve --body "Automated approval for release ${VERSION_NUMBER}"
gh pr merge "$PR_NUMBER" --delete-branch
