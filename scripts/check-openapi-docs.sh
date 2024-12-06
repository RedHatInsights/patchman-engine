#!/usr/bin/env bash

APIVERS="v3 admin"
declare -A OPENAPI_COPY
for APIVER in $APIVERS; do
  OPENAPI_COPY["$APIVER"]=$(mktemp -t openapi.json.XXX)
  cp docs/$APIVER/openapi.json ${OPENAPI_COPY["$APIVER"]}
done

./scripts/generate_docs.sh

for APIVER in ${!OPENAPI_COPY[@]}; do
  diff docs/$APIVER/openapi.json ${OPENAPI_COPY["$APIVER"]}
  rc+=$?
  if [ $rc -gt 0 ]; then
    echo "docs/$APIVER/openapi.json different from file generated with './scripts/generate_docs.sh'!"
  else
    echo "docs/$APIVER/openapi.json consistent with generated file."
  fi

  rm ${OPENAPI_COPY["$APIVER"]}
done
exit $rc
