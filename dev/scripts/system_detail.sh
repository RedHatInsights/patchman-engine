#!/bin/bash

IDENTITY="$($(dirname "$0")/identity.sh)"
UUID=${1:-00000000-0000-0000-0000-000000000001}

curl -v -H "x-rh-identity: $IDENTITY" http://localhost:8080/api/patch/v2/systems/$UUID | python3 -m json.tool
