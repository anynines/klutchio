#!/usr/bin/env bash

# Copyright 2021 The Kube Bind Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

export GOPATH=$(go env GOPATH)

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; go list -f '{{.Dir}}' -m k8s.io/code-generator)}

# Go 1.24 enables VCS stamping by default, but module cache paths used by
# kube_codegen.sh are not always VCS worktrees.
if [[ "${GOFLAGS:-}" != *"-buildvcs=false"* ]]; then
  export GOFLAGS="${GOFLAGS:-} -buildvcs=false"
fi

source "${CODEGEN_PKG}/kube_codegen.sh"

kube::codegen::gen_helpers \
  --boilerplate "${SCRIPT_ROOT}/hack/boilerplate/boilerplate.generatego.txt" \
  "${SCRIPT_ROOT}/pkg/apis"

kube::codegen::gen_client \
  --with-watch \
  --boilerplate "${SCRIPT_ROOT}/hack/boilerplate/boilerplate.generatego.txt" \
  --output-dir "${SCRIPT_ROOT}/pkg/client" \
  --output-pkg github.com/anynines/klutchio/bind/pkg/client \
  "${SCRIPT_ROOT}/pkg/apis"

kube::codegen::gen_helpers \
  --boilerplate "${SCRIPT_ROOT}/hack/boilerplate/boilerplate.generatego.txt" \
  "${SCRIPT_ROOT}/contrib/example-backend/apis"

kube::codegen::gen_client \
  --with-watch \
  --boilerplate "${SCRIPT_ROOT}/hack/boilerplate/boilerplate.generatego.txt" \
  --output-dir "${SCRIPT_ROOT}/contrib/example-backend/client" \
  --output-pkg github.com/anynines/klutchio/bind/contrib/example-backend/client \
  "${SCRIPT_ROOT}/contrib/example-backend/apis"
