#!/usr/bin/env bash

set -e -o pipefail # stop on error

COMPONENT=$1
# This script is launched inside the /go/src/app working directory
echo "Running in $(pwd) as $(id)"
if [[ -n $GORUN ]]; then
  # Running using 'go run'
  exec go run main.go $COMPONENT
else
  exec ./main $COMPONENT
fi
