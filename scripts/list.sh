#!/bin/bash

# ./list.sh

curl -v -XGET http://localhost:8080/samples | python -m json.tool
