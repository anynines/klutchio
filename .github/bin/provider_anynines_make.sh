#! /usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <make-target> <ecr-registry-address> <version-nr>"
  exit 1
fi

MAKE_TARGET="$1"
ECR_REGISTRY_ADDRESS="$2"
VERSION_NR="$3"

"$(dirname "$0")/login_ecr_public.sh" "${ECR_REGISTRY_ADDRESS}"

git submodule update --init --recursive
make "${MAKE_TARGET}" IMAGETAG="${VERSION_NR}" dataservicesConfigVersion="${VERSION_NR}"
