#!/bin/bash

source "$(dirname $(realpath "$0"))/env.sh"

curl -v -H "x-rh-identity: $IDENTITY" -XGET http://localhost:8080/api/patch/v1/systems/00000000-0000-0000-0000-000000000001 | python3 -m json.tool
