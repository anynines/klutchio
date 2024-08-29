#!/bin/bash

set -euo pipefail

verbose=${TEST_VERBOSE:-false}

try_apply() {
    local manifest=$1 output=$2
    if kubectl apply --dry-run=server -f $manifest >$output 2>&1 ; then
	echo "valid"
    else
	echo "invalid"
    fi
    if [[ $verbose == true ]]; then
	echo "~~~~~~~~~~~~~~~" >&2
	cat $output >&2
    fi
}

run_case() {
    local testpath=$1
    local testname=$(basename $(dirname $testpath))/$(basename $testpath)
    local expect=$(echo $testname | cut -d - -f 2)
    local testoutput=$TMPDIR/testoutput
    local result=$(try_apply $testpath $testoutput)

    if [[ $expect != $result ]]; then
	echo "🔴 ${testname}: expected ${expect}, got ${result}"
	echo "While trying to apply manifest $testpath. Output:"
	cat $testoutput
	return 1
    else
	echo "🟢 ${testname}"
	return 0
    fi
}

status=0

if [[ $# > 0 ]]; then
    cases="$@"
else
    base=$(dirname $0)
    cases=$(ls $base/*/create-{valid,invalid}-*.yaml)
fi

for testpath in $cases; do
    run_case $testpath || status=1
done

if [[ $status == 0 ]]; then
    echo "✅ All 'create' tests succeeded"
else
    echo "❌ Some of the tests failed. Check the output above for details."
fi

exit $status
