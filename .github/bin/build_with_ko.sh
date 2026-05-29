#! /usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <path-to-code> <image-repo> <image-tag>"
  exit 1
fi

PATH_TO_CODE="$1"
IMAGE_REPO="$2"
IMAGE_TAG="$3"

"$(dirname "$0")/login_ecr_public.sh" "$IMAGE_REPO"

KO_DOCKER_REPO="$IMAGE_REPO" ko build \
  "$PATH_TO_CODE" --bare -t "${IMAGE_TAG}" \
  --platform=linux/amd64,linux/arm64
