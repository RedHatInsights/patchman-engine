#!/bin/bash

# Usage:
# ./generate_docs.sh

DOCS_TMP_DIR=/tmp
CONVERT_URL="https://converter.swagger.io/api/convert"
VERSION=$(cat VERSION)

# Create temporary swagger 2.0 definition
swag init --output $DOCS_TMP_DIR --exclude turnpike --generalInfo manager/manager.go
swag init --output $DOCS_TMP_DIR/admin --dir turnpike --generalInfo admin_api.go

convert_doc() {
  local in=$1
  local out=$2
  curl -X "POST" -H "accept: application/json" -H  "Content-Type: application/json" \
  -d @$DOCS_TMP_DIR/$in $CONVERT_URL \
  | python3 -m json.tool | sed "s/{{.Version}}/$VERSION/" \
  > $out
}

# Perform conversion
convert_doc swagger.json docs/v2/openapi.json

# Convert admin spec
convert_doc admin/swagger.json docs/admin/openapi.json
