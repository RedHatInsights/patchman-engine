#!/bin/bash

rc=0

DEV=docker-compose.yml
PROD=docker-compose.prod.yml
# Check consistency of docker-compose.yml and docker-compose.yml
sed \
    -e "s|INSTALL_TOOLS=yes|INSTALL_TOOLS=no|" \
    -e "s|target: buildimg|target: runtimeimg|" \
    -e "/ - \.\/conf\/gorun.env/ d" \
    -e "/  \(db_admin\|db_feed\|manager\|listener\|evaluator_recalc\|evaluator_upload\|vmaas_sync\|admin\):/,/^$/ {
      s/- \.\/:\/go\/src\/app/- \.\/dev:\/go\/src\/app\/dev\n\
      - .\/dev\/database\/secrets:\/opt\/postgresql\n\
      - \.\/dev\/kafka\/secrets:\/opt\/kafka/
      }" \
    "$DEV" | diff -u - "$PROD"
rc=$?
if [ $rc -gt 0 ]; then
  echo "$DEV and $PROD are too different!"
else
  echo "$DEV and $PROD are OK"
fi
echo

exit $rc
