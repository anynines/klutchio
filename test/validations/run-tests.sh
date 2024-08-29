#!/bin/bash

set -euo pipefail

status=0

base=$(dirname $0)
$base/run-create-tests.sh || status=1
$base/run-update-tests.sh || status=1

exit $status
