#!/usr/bin/env bash

# Please set ProviderNameLower & ProviderNameUpper environment variables before running this script.
# See: https://github.com/crossplane/terrajet/blob/main/docs/generating-a-provider.md
set -euo pipefail

APIVERSION="${APIVERSION:-v1alpha1}"
echo "Adding type ${KIND} to group ${GROUP} with version ${APIVERSION}"

export GROUP
export KIND
export APIVERSION
export PROVIDER
export PROJECT_REPO

kind_lower=$(echo "${KIND}" | tr "[:upper:]" "[:lower:]")
group_lower=$(echo "${GROUP}" | tr "[:upper:]" "[:lower:]")

mkdir -p "apis/${group_lower}/${APIVERSION}"
${GOMPLATE} < "hack/helpers/apis/GROUP_LOWER/GROUP_LOWER.go.tmpl" > "apis/${group_lower}/${group_lower}.go"
${GOMPLATE} < "hack/helpers/apis/GROUP_LOWER/APIVERSION/KIND_LOWER_types.go.tmpl" > "apis/${group_lower}/${APIVERSION}/${kind_lower}_types.go"
${GOMPLATE} < "hack/helpers/apis/GROUP_LOWER/APIVERSION/doc.go.tmpl" > "apis/${group_lower}/${APIVERSION}/doc.go"
${GOMPLATE} < "hack/helpers/apis/GROUP_LOWER/APIVERSION/groupversion_info.go.tmpl" > "apis/${group_lower}/${APIVERSION}/groupversion_info.go"

mkdir -p "internal/controller/${kind_lower}"
${GOMPLATE} < "hack/helpers/controller/KIND_LOWER/KIND_LOWER.go.tmpl" > "internal/controller/${kind_lower}/${kind_lower}.go"
${GOMPLATE} < "hack/helpers/controller/KIND_LOWER/KIND_LOWER_test.go.tmpl" > "internal/controller/${kind_lower}/${kind_lower}_test.go"



