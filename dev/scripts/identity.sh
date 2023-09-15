#!/bin/bash

# This identity contains account_name = "0"
ACCOUNT_NUMBER=${1:-"org_1"}

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
        "account_number": "$ACCOUNT_NUMBER",
        "org_id": "$ACCOUNT_NUMBER",
        "auth_type": "basic-auth",
        "type": "User",
        "user": {
            "username": "jdoe@acme.com",
            "email": "jdoe@acme.com",
            "first_name": "john",
            "last_name": "doe",
            "is_active": true,
            "is_org_admin": false,
            "is_internal": false,
            "locale": "en_US"
        },
        "internal": {
            "org_id": "$ACCOUNT_NUMBER",
            "auth_type": "basic-auth",
            "auth_time": 6300
        }
    },
    "entitlements": {
        "insights": {
            "is_entitled": true
        },
        "smart_management": {
            "is_entitled": true
        }
    }
}
IDENTITY
echo
