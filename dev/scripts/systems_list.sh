#!/bin/bash

curl -v -XGET http://localhost:8080/api/patch/v1/systems | python -m json.tool
