#!/bin/bash

set -e -o pipefail

echo "Running in $(pwd) as $(id)"
./database_admin/update.sh

exec sleep infinity
