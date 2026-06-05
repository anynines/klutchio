#! /usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
    echo "Usage: $0 <branch-name>"
    exit 1
fi

BRANCH_NAME=$1

OUTPUT="$(gh pr create \
    --title "chore: update via gh cli" \
    --body "This PR was created using the GitHub CLI." \
    --base main-test \
    --head "$BRANCH_NAME")"

echo "$OUTPUT"

PR_NUMBER="${OUTPUT##*/}"

echo "Waiting for PR #$PR_NUMBER to be merged..."
while true; do
    MERGED_AT=$(gh pr view "$PR_NUMBER" --json mergedAt -q '.mergedAt')
    if [[ "$MERGED_AT" != "null" && "$MERGED_AT" != "" ]]; then
        echo "PR merged!"
        break
    fi
    echo "Not merged yet. Waiting 10s..."
    sleep 10
done
