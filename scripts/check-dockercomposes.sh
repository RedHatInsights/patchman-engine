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
    "$DEV" | diff -u - "$PROD"
diff_rc=$?
if [ $diff_rc -gt 0 ]; then
  echo "$DEV and $PROD are too different!"
else
  echo "$DEV and $PROD are OK"
fi
echo

exit $rc
