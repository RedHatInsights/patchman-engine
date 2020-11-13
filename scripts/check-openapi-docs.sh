#!/bin/bash

cp docs/openapi.json docs/openapi1.json
./scripts/generate_docs.sh --keep
diff docs/openapi.json docs/openapi1.json
rc=$?
if [ $rc -gt 0 ]; then
  echo "docs/openapi.json different from file generated with './scripts/generate_docs.sh'!"
else
  echo "docs/openapi.json consistent with generated file."
fi

exit $rc
