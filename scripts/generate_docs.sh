#!/usr/bin/bash


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