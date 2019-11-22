#!/bin/bash

# ./graphql.sh

curl -vg -XGET 'http://localhost:8080/graphql?query={hosts(limit:3){id,request,checksum,updated}}' | python -m json.tool
