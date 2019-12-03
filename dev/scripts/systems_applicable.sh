#!/bin/bash

curl -v -XGET http://localhost:8080/api/patch/v1/advisories/$1/systems | python -m json.tool
