#!/usr/bin/bash

# Usage:
# ./generate_docs.sh --increment
#         Reads curent X.Y.Z version (last git tag) and increments Z part
# ./generate_docs.sh --keep
#         Keeps current version untouched
# ./generate_docs.sh --release "A.B.C"
#         Sets version exactly to "A.B.C"
#

RELEASE=""
case "$1" in
  -i|--increment)
        CURRENT_RELEASE=$(git tag | tail -n1)
        RELEASE="${CURRENT_RELEASE%.*}.$((${CURRENT_RELEASE##*.}+1))"
        ;;
  -k|--keep) # don't change anything, just keep current release
        ;;
  -r|--release) RELEASE=$2
        ;;
  *) >&2 echo "Usage: $0 [ [-k|--keep] | [-i|--increment] | [-r|--release] <release> ]"
        exit 1
        ;;
esac

if [ -n "$RELEASE" ] ; then
  # Substitute version
  sed -i "s|\(// @version \).*$|\1 $RELEASE|;" manager/manager.go
  sed -i 's|^\(var ENGINEVERSION = "\)[^"]*\("\)$|'"\1$RELEASE\2|;" base/metrics/metrics.go
fi

DOCS_TMP_DIR=/tmp
CONVERT_URL="https://converter.swagger.io/api/convert"

# Create temporary swagger 2.0 definition
swag init --output $DOCS_TMP_DIR --generalInfo manager/manager.go

# We can run the converter container ourelves if we want to
#PID=$(docker run -d -p 28080:8080 --name swagger-converter swaggerapi/swagger-converter:v1.0.2)

# Wait for converter to be ready
until curl $CONVERT_URL > /dev/null 2> /dev/null; do
  sleep 2
done


# Perform conversion
curl -X "POST" -H "accept: application/json" -H  "Content-Type: application/json" \
  -d @$DOCS_TMP_DIR/swagger.json $CONVERT_URL \
  | python3 -m json.tool \
  > docs/openapi.json


if [ ! -z "$PID" ]
then
  # Cleanup
  docker container rm -f "$PID"
fi
