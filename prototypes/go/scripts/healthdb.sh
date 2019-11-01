#!/bin/bash

# ./healthdb.sh

curl -v -XGET http://localhost:8080/db_health && echo
