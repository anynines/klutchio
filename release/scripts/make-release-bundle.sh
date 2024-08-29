#!/bin/bash

set -euo pipefail

if [[ $# != 1 ]]; then
    echo "Usage: $0 <release-version>" >&2
    exit 1
fi

# Version of the resulting release
RELEASE_VERSION="$1"

# All of these variables must be set:
for var in CONFIG_VERSION PROVIDER_ANYNINES_VERSION PROVIDER_KUBERNETES_VERSION KUBE_BIND_BACKEND_VERSION; do
    if [ -z "$(eval 'echo ${'$var'+x}')" ]; then
	echo "Missing variable: ${var}"
	exit 1
    fi
done

base=$(realpath "$(dirname $0)/..")
output="${base}/output"

# These bundles are always generated
bundles="install install-dev update"

# Generate configure bundle only for major and minor releases (patchlevel is 0)
if echo "$RELEASE_VERSION" | grep -Eq '^\d+\.\d+\.0'; then
    bundles="${bundles} configure"
fi

mkdir -p "$output"

patch_path="${base}/patches/release.yaml"
patch_template_path="${patch_path}.template"
bundle_tmp_path="${output}/bundle.tmp.yaml"

cleanup() {
    rm -f "$patch_path"
    rm -f "$bundle_tmp_path"
}

trap cleanup EXIT

# Generate release patch from template
envsubst < "$patch_template_path" > "$patch_path"

for bundle in $bundles ; do
    output_path="${output}/${bundle}-v${RELEASE_VERSION}.yaml"

    # Produce bundle
    kustomize build --load-restrictor LoadRestrictionsNone "${base}/bundles/${bundle}" > "$bundle_tmp_path"

    # The bases use "PLACEHOLDER" as a generic marker for values that the release
    # patch is supposed to supply. If the output still contains that marker, we've
    # made a mistake somewhere.
    if grep -q PLACEHOLDER "$bundle_tmp_path"; then
	broken_path="${output}/${bundle}-broken.yaml"
	cp "$bundle_tmp_path" "$broken_path"
	echo "BUG: bundle still contains PLACEHOLDER, check ${broken_path}" >&2
	exit 1
    fi

    cp "$bundle_tmp_path" "$output_path"
    echo "Produced bundle: ${output_path}"
done
