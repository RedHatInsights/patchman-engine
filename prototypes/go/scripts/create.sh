#!/bin/bash

# ./create.sh 1 1.23

curl -v -u admin:passwd -XGET http://localhost:8080/private/create?id=${1}&value=${2}
