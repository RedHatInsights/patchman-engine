#!/bin/bash

curl -v -XGET http://localhost:8080/api/patch/v1/systems$1 | python -m json.tool
