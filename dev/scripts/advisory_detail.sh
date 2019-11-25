#!/bin/bash

curl -v -XGET http://localhost:8080/api/patch/v1/advisories/$1 | python -m json.tool
