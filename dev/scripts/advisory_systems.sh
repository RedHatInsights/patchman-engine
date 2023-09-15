#!/bin/bash

IDENTITY="$($(dirname "$0")/identity.sh)"
ADVISORY=${1:-RH-1}

curl -v -H "x-rh-identity: $IDENTITY" http://localhost:8080/api/patch/v1/advisories/$ADVISORY/systems | python3 -m json.tool
