#!/bin/bash

set -euo pipefail

## This script prints a summary of the contents of a YAML bundle

if [[ $# != 1 ]]; then
    echo "Usage: $0 <yaml-bundle>" >&2
    exit 1
fi

bundle_file="$1"

yq -N '.apiVersion + "/" + .kind + ": " + .metadata.name' < "$bundle_file" | sort
