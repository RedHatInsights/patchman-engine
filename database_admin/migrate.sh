#!/bin/bash

set -o pipefail

/migrate -source file:///database_admin/migrations -database postgres://$DB_HOST/$DB_NAME?sslmode=$DB_SSLMODE up
