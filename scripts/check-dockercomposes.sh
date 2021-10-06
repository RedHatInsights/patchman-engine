#!/bin/bash

rc=0

DEV=docker-compose.yml
PROD=docker-compose.prod.yml
# Check consistency of docker-compose.yml and docker-compose.yml
sed \
    -e "s|INSTALL_TOOLS=yes|INSTALL_TOOLS=no|" \
    -e "s|target: buildimg|target: runtimeimg|" \
    -e "/ - \.\/conf\/gorun.env/ d" \
    -e "/    volumes:/,+1 { N;}; /- \.\/:\/go\/src\/app/ d" \
    -e "/ - BUILDIMG=centos:8/ d" \
    -e "/ - RUNIMG=centos:8/ d" \
    "$DEV" | diff -u - "$PROD"
rc=$?
if [ $rc -gt 0 ]; then
  echo "$DEV and $PROD are too different!"
else
  echo "$DEV and $PROD are OK"
fi
echo

exit $rc
