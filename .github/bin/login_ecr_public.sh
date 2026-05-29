#! /usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
    echo "Usage: $0 <ecr-public-registry-url>"
    exit 1
fi

aws ecr-public get-login-password --region us-east-1 |
    docker login --username AWS \
        --password-stdin "$1"
