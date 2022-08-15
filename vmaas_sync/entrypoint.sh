#!/usr/bin/env bash


if [ -z "$VMAAS_WS_ADDRESS" ]; then
  echo "Set VMASS_WS_ADDRESS env variable to point to VMaaS"
  exit 1
fi

echo "Running in $(pwd) as $(id)"
exec ./scripts/entrypoint.sh job vmaas_sync
