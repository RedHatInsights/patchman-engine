#!/bin/bash

APIVER=v3
OPENAPI_COPY=$(mktemp -t openapi.json.XXX)
cp docs/$APIVER/openapi.json $OPENAPI_COPY
./scripts/generate_docs.sh
diff docs/$APIVER/openapi.json $OPENAPI_COPY
rc=$?
if [ $rc -gt 0 ]; then
  echo "docs/$APIVER/openapi.json different from file generated with './scripts/generate_docs.sh'!"
else
  echo "docs/$APIVER/openapi.json consistent with generated file."
fi

rm $OPENAPI_COPY
exit $rc
