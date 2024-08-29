#!/bin/bash

set -euo pipefail

verbose=${TEST_VERBOSE:-false}

testns="validation-update-test"

logpath=$(dirname $0)/update-test.log

echo "Logging to $logpath"

# truncate
echo "" >$logpath

try_patch() {
    local object_name=$1 patch=$2 output=$3
    if kubectl -n $testns patch --dry-run=server --type=merge $object_name -p "$patch" >$output 2>&1; then
	echo "valid"
    else
	echo "invalid"
    fi
    if [[ $verbose == true ]]; then
	echo "~~~~~~~~~~~~~~~" >&2
	cat $output >&2
    fi
}

run_patch_case() {
    local expect=$1 testname=$2 object_name=$3 index=$4 patch="$5"

    echo "TEST: ${testname}/${expect}_patches[${index}]" >>$logpath

    local patchout=$TMPDIR/patchoutput
    local result=$(try_patch "$object_name" "$patch" "$patchout")

    cat $patchout >>$logpath

    if [[ $expect != $result ]]; then
	echo "🔴 ${testname}/${expect}_patches[${index}]: expected ${expect}, got ${result}"
	echo "While trying to update ${object_name} with ${patch}:"
	cat $patchout
	return 1
    else
	echo "🟢 ${testname}/${expect}_patches[${index}]"
	return 0
    fi
}

run_case() {
    local testpath=$1
    local object_name=$(cat $testpath | yq .base | kubectl -n $testns apply -f - -oname)
    local testname=$(basename $(dirname $testpath))/$(basename $testpath)
    local status=0

    local valid_patch_count=$(cat $testpath | yq '.valid_patches | length')
    if [[ $valid_patch_count > 0 ]]; then
	for i in $(seq 0 $(expr $valid_patch_count - 1)); do
	    local patch=$(cat $testpath | yq ".valid_patches[${i}]" -ojson)
	    run_patch_case valid $testname $object_name $i "$patch" || status=1
	done
    fi

    local invalid_patch_count=$(cat $testpath | yq '.invalid_patches | length')
    if [[ $invalid_patch_count > 0 ]]; then
	for i in $(seq 0 $(expr $invalid_patch_count - 1)); do
	    local patch=$(cat $testpath | yq ".invalid_patches[${i}]" -ojson)
	    run_patch_case invalid $testname $object_name $i "$patch" || status=1
	done
    fi

    kubectl -n $testns delete $object_name --wait=false
    return $status
}

kubectl create namespace $testns

cleanup() {
    kubectl delete namespace $testns
}
trap cleanup EXIT

if [[ $# > 0 ]]; then
    cases="$@"
else
    base=$(dirname $0)
    cases=$(ls $base/*/update-*.yaml)
fi

status=0

for testpath in $cases; do
    run_case $testpath || status=1
done

if [[ $status == 0 ]]; then
    echo "✅ All 'update' tests succeeded"
else
    echo "❌ Some of the tests failed. Check the output above, and consult $logpath for more details"
fi

exit $status
