#!/usr/bin/env bash

COMPONENT=$1
# This script is launched inside the /go/src/app working directory
if [[ -n $GORUN ]]; then
  # Running using 'go run'
  exec ./scripts/wait-for-services.sh go run main.go $COMPONENT
else
  exec ./scripts/wait-for-services.sh ./main $COMPONENT
fi
