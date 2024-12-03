#!/bin/bash

# This identity contains account_name = "0"
SYSTEM_SUBSRIPTION_UUID=${1:-"cccccccc-0000-0000-0001-000000000004"}
ORG_ID=${2:-"org_1"}

encode() {
        local input=$(</dev/stdin)
        if type -p jq >/dev/null ; then
          input=$(jq -cM <<<"$input")
        fi
        base64 -w 0 - <<<"$input"
}

encode <<IDENTITY
{
    "identity": {
        "org_id": "$ORG_ID",
        "auth_type": "cert-auth",
        "type": "System",
        "system": {
            "cert_type": "system",
            "cn": "$SYSTEM_SUBSRIPTION_UUID"
        },
        "internal": {
            "org_id": "$ORG_ID",
            "auth_type": "cert-auth",
            "auth_time": 6300
        }
    },
    "entitlements": {
        "insights": {
            "is_entitled": true
        }
    }
}
IDENTITY

echo
