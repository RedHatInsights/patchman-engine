#!/bin/bash

OPENAPI_COPY=$(mktemp -t openapi.json.XXX)
cp docs/v2/openapi.json $OPENAPI_COPY
./scripts/generate_docs.sh
diff docs/v2/openapi.json $OPENAPI_COPY
rc=$?
if [ $rc -gt 0 ]; then
  echo "docs/v2/openapi.json different from file generated with './scripts/generate_docs.sh'!"
else
  echo "docs/v2/openapi.json consistent with generated file."
fi

rm $OPENAPI_COPY
exit $rc
