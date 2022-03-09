#!/usr/bin/bash

# Usage:
# ./generate_docs.sh

DOCS_TMP_DIR=/tmp
CONVERT_URL="https://converter.swagger.io/api/convert"

# Create temporary swagger 2.0 definition
swag init --output $DOCS_TMP_DIR --generalInfo manager/manager.go

# Perform conversion
curl -X "POST" -H "accept: application/json" -H  "Content-Type: application/json" \
  -d @$DOCS_TMP_DIR/swagger.json $CONVERT_URL \
  | python3 -m json.tool \
  > docs/openapi.json
