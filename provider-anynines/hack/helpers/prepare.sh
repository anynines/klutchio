#!/usr/bin/env bash

# Please set ProviderNameLower & ProviderNameUpper environment variables before running this script.
# See: https://github.com/crossplane/terrajet/blob/main/docs/generating-a-provider.md
set -euo pipefail

ProviderNameUpper=${PROVIDER}
ProviderNameLower=$(echo "${PROVIDER}" | tr "[:upper:]" "[:lower:]")

git rm -r apis/sample
git rm -r internal/controller/mytype

REPLACE_FILES='./* ./.github :!build/** :!go.* :!hack/**'
# shellcheck disable=SC2086
git grep -l 'template' -- ${REPLACE_FILES} | xargs sed -i.bak "s/template/${ProviderNameLower}/g"
# shellcheck disable=SC2086
git grep -l 'Template' -- ${REPLACE_FILES} | xargs sed -i.bak "s/Template/${ProviderNameUpper}/g"
# We need to be careful while replacing "template" keyword in go.mod as it could tamper
# some imported packages under require section.
sed -i.bak "s/provider-template/provider-${ProviderNameLower}/g" go.mod

# Clean up the .bak files created by sed
git clean -fd

git mv "apis/template.go" "apis/${ProviderNameLower}.go"
git mv "internal/controller/template.go" "internal/controller/${ProviderNameLower}.go"
