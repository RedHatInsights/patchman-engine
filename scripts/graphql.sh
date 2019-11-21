#!/bin/bash

# ./graphql.sh

curl -vg -XGET 'http://localhost:8080/graphql?query={host(id:1){id,request,checksum,updated}}' | python -m json.tool
