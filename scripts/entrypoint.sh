#!/usr/bin/env bash

set -e -o pipefail # stop on error

COMPONENT=$1
# This script is launched inside the /go/src/app working directory
echo "Running in $(pwd) as $(id)"
exec ${GORUN:+go run} ./main${GORUN:+.go} $COMPONENT
