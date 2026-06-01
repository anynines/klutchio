#! /usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 1 ]]; then
    echo "Usage: $0 <version_nr>"
    exit 1
fi

VERSION_NR="$1"
HEADLINE="[${VERSION_NR:1}] - $(date "+%Y-%m-%d")"

add_version_section() {
    if grep -x -q -F "## $HEADLINE" "./CHANGELOG.md"; then
        echo "Changelog already contains a section for $VERSION_NR. Skipping..."
        return
    fi

    echo "Replacing 'Unreleased' changelog section header with" \
        "header '$HEADLINE'..."
    sed -E 's/^## Unreleased$/'"## $HEADLINE/g" \
        "./CHANGELOG.md" >"./CHANGELOG.md.tmp"
    mv "./CHANGELOG.md.tmp" "./CHANGELOG.md"
}

recreate_unreleased_section() {
    if grep -q "^## Unreleased$" "CHANGELOG.md"; then
        echo 'Changelog already contains an "Unreleased" section. Skipping" \
        "recreation of "Unreleased" section...'
        return
    fi

    echo 'Recreating "Unreleased" section in Changelog...'
    (
        echo "# CHANGELOG"
        echo
        echo "## Unreleased"
        tail -n +2 "CHANGELOG.md"
    ) >"CHANGELOG.md.tmp"
    mv "CHANGELOG.md.tmp" "CHANGELOG.md"
}

add_version_section
recreate_unreleased_section
