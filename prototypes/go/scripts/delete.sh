#!/bin/bash

# ./delete.sh 1

curl -v -u admin:passwd -XGET http://localhost:8080/delete/create?id=${1}
