#!/bin/bash

# ./get_host.sh 1

curl -v -XGET http://localhost:8080/hosts/$1
