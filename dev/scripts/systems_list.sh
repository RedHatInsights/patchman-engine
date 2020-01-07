#!/bin/bash

source "$(dirname $(realpath "$0"))/env.sh"

curl -v -H "x-rh-identity: $IDENTITY" -XGET http://localhost:8080/api/patch/v1/systems$1 | python -m json.tool
