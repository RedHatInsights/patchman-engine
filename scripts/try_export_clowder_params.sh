#!/bin/bash
# Load Clowder params from json and export them as a environment variables.

# Use go app command to print Clowder params
function print_clowder_params() {
  if [[ -n $GORUN ]]; then
    go run main.go print_clowder_params
  else
    ./main print_clowder_params
  fi
}

# Detect params, it should be printed in 'PARAM=VALUE' format
CLOWDER_PARAMS=$(print_clowder_params | grep '=' || true) # use '|| true' not to stop on empty match

# Enable to show Clowder vars in logs
if [[ -n $SHOW_CLOWDER_VARS ]]; then
  echo $CLOWDER_PARAMS
fi

# Export Clowder params if any found
if [[ ! -z $CLOWDER_PARAMS ]]; then
  echo "Clowder params found, setting..."
  export $CLOWDER_PARAMS
else
  echo "No Clowder params found"
fi
