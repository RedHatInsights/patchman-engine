#!/bin/bash

curl -v -XGET http://localhost:8080/api/patch/v1/advisories | python -m json.tool
