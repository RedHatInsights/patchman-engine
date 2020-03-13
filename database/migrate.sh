#!/bin/bash

set -o pipefail

/migrate -source file:///database/migrations -database postgres://$DB_HOST/$DB_NAME?sslmode=disable up
