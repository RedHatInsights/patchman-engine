#!/bin/bash

IDENTITY="$($(dirname "$0")/identity.sh)"

curl -v -H "x-rh-identity: $IDENTITY" -XGET http://localhost:8080/api/patch/v1/advisories/RH-1/systems | python3 -m json.tool
