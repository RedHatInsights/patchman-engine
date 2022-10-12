#!/bin/bash

if [[ -n $DB_HOST ]] ; then
    ./dev/scripts/wait-for-services.sh
fi

exec ./scripts/entrypoint.sh "$@"
