#!/bin/bash

IDENTITY="$($(dirname "$0")/identity.sh)"

curl -v -H "x-rh-identity: $IDENTITY" http://localhost:8080/api/patch/v3/advisories/RH-1/systems | python3 -m json.tool
