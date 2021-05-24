#!/usr/bin/env bash


if [ -z "$VMAAS_WS_ADDRESS" ]; then
  echo "Set VMASS_WS_ADDRESS env variable to point to VMaaS"
  exit 1
fi

# Replace websocket prefixes for curl
search1="ws://"
replace1="http://"
search2="wss://"
replace2="https://"

TEST_ADDR=${VMAAS_WS_ADDRESS/$search1/$replace1}
TEST_ADDR=${TEST_ADDR/$search2/$replace2}

until curl -v $TEST_ADDR > /dev/null 2> /dev/null; do
  echo "Waiting for VMaaS websocket to be available"
  sleep 1
done

echo "VMaaS websocket is up"

echo "Running in $(pwd) as $(id)"
exec ./scripts/entrypoint.sh vmaas_sync
