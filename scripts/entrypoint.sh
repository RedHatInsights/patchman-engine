#!/usr/bin/env bash

set -e -o pipefail # stop on error

source ./scripts/try_export_clowder_params.sh

COMPONENT=$1
# This script is launched inside the /go/src/app working directory
echo "Running in $(pwd) as $(id)"
if [[ -n $GORUN ]]; then
  # Running using 'go run'
  exec $CMD_WRAPPER ./scripts/wait-for-services.sh go run main.go $COMPONENT
else
  exec $CMD_WRAPPER ./scripts/wait-for-services.sh ./main $COMPONENT
fi
