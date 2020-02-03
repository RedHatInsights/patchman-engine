#!/usr/bin/env bash

# This script is launched inside the /go/src/app working directory
if [ -n $GORUN ]; then
  # Running using 'go run'
  ./scripts/wait-for-services.sh go run main.go manager
else
  ./scripts/wait-for-services.sh ./main manager
fi
