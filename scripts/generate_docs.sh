#!/bin/bash

# Usage:
# ./generate_docs.sh

DOCS_TMP_DIR=/tmp
CONVERT_URL="https://converter.swagger.io/api/convert"

# Create temporary swagger 2.0 definition
swag init --output $DOCS_TMP_DIR --exclude turnpike --generalInfo manager/manager.go
swag init --output $DOCS_TMP_DIR/admin --dir turnpike --generalInfo admin_api.go

# Perform conversion
curl -X "POST" -H "accept: application/json" -H  "Content-Type: application/json" \
  -d @$DOCS_TMP_DIR/swagger.json $CONVERT_URL \
  | python3 -m json.tool \
  > docs/v2/openapi.json

# Convert admin spec
curl -X "POST" -H "accept: application/json" -H  "Content-Type: application/json" \
  -d @$DOCS_TMP_DIR/admin/swagger.json $CONVERT_URL \
  | python3 -m json.tool \
  > docs/admin/openapi.json
