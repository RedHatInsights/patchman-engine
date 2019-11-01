#!/bin/bash

# ./health.sh

curl -v -XGET http://localhost:8080/health && echo
