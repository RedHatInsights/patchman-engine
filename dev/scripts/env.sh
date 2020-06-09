# This identity contains account_name = "0"

ACCOUNT_NUMER=${1:-"1"}
JSON='{"entitlements":{"smart_management":{"is_entitled":true}},"identity":{"account_number":"'$ACCOUNT_NUMER'","type":"User"}}'
IDENTITY=$(echo "$JSON" | base64 -w 0 -)
