#!/usr/bin/env sh

set -eux

if [ $# -eq 0 ]
then
    echo "usage: $0 <new version number>"
    exit 1
fi

find ./docs \
    -type f -exec \
    gsed -i "s,\(public.ecr.aws/w5n9a2g2/klutch/dataservices:v\)\([[:digit:]]\.[[:digit:]]\.[[:digit:]]\),\1$1,g" {} \;

find ./docs \
    -type f -exec \
    gsed -i "s,\(anynines-artifacts.s3.eu-central-1.amazonaws.com/central-management/:v\)\([[:digit:]]\.[[:digit:]]\.[[:digit:]]\),\1$1,g" {} \;
