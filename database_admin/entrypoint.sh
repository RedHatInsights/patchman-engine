#!/bin/bash

set -e -o pipefail

./database_admin/update.sh

exec sleep infinity
