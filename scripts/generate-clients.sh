#!/bin/bash

#CONTAINER=openapitools/openapi-generator-cli
# We temporarily use image built from https://github.com/semtexzv/openapi-generator
# until the https://github.com/OpenAPITools/openapi-generator/pull/4664 is merged
CONTAINER=semtexzv/openapi-generator-cli:latest

# For now, we skip the dateTime parsing because of incompatible formats ( inventory does not produce fully compliant
# RFC 3339 datetimes

function generate_client() {
  NAME=$1
  SPEC_URL=$2

  curl $SPEC_URL > /tmp/openapi.json

  HERE=$(pwd)
  docker run --rm -v ${HERE}:/local:z -v /tmp:/tmp -v /etc/passwd:/etc/passwd -u `id -u`:`id -g` $CONTAINER generate \
      -i /tmp/openapi.json \
      -g go \
      --api-package $NAME \
      -p packageName=$NAME,isGoSubmodule=true \
      --git-host app --git-user-id _generated --git-repo-id cmsfr \
      --type-mappings DateTime=string \
      -o /local/_generated/cmsfr/$NAME
}

generate_client inventory "https://ci.cloud.redhat.com/api/inventory/v1/openapi.json"
#generate_client remediations "https://ci.cloud.redhat.com/api/remediations/v1/openapi.json"
generate_client vmaas "https://webapp-vmaas-ci.5a9f.insights-dev.openshiftapps.com/api/openapi.json"
