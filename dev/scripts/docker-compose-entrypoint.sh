#!/bin/bash

if [[ -n $DB_HOST ]] ; then
    ./dev/scripts/wait-for-services.sh
fi

if [[ -n $KAFKA_READY_ADDRESS ]] ; then
   ./dev/scripts/wait-for-kafka.sh
fi

exec ./scripts/entrypoint.sh "$@"
