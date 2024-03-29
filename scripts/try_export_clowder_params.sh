#!/bin/bash
# Load Clowder params from json and export them as a environment variables.

# Use go app command to print Clowder params
function print_clowder_params() {
  ${GORUN:+go run} ./main${GORUN:+.go} print_clowder_params
}

if [[ -n $ACG_CONFIG ]] ; then
  # clowder is enabled
  CLOWDER_PARAMS=$(print_clowder_params)

  # Enable to show Clowder vars in logs
  if [[ -n $SHOW_CLOWDER_VARS ]]; then
    echo $CLOWDER_PARAMS
  fi

  echo "Clowder params found, setting..."
  export $CLOWDER_PARAMS
else
  echo "No Clowder params found"
fi
