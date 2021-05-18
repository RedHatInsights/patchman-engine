#!/bin/bash

set -e -o pipefail # stop on error

source ./scripts/try_export_clowder_params.sh

echo "Running in $(pwd) as $(id)"
./database_admin/update.sh

exec sleep infinity
