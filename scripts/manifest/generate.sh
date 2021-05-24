#!/bin/sh

# ./generate.sh <MANIFEST_PATH> <PREFIX> <BASE_RPM_LIST> <FINAL_RPM_LIST> <PYTHON-CMD-OPTIONAL> <GO-SUM-PATH>
# Example:
# ./generate.sh manifest_webapp.txt my-service /tmp/base_rpm_list.txt /tmp/final_rpm_list.txt python /app/go.sum
# cat manifest_webapp.txt

MANIFEST_PATH=$1
PREFIX=$2
BASE_RPM_LIST=$3
FINAL_RPM_LIST=$4
PYTHON=$5
GO_SUM=$6

grep -v -f ${BASE_RPM_LIST} ${FINAL_RPM_LIST} > ${MANIFEST_PATH}

## Write Python packages if python set.
if [[ ! -z "$PYTHON" ]]
then
    "$PYTHON" -m pip freeze | sort | \
    sed -e "s/^/$PYTHON-/; # add 'python' prefix" \
        -e "s/==/-/;       # replace '==' with '-'" \
        -e "s/\$/.pipfile/ # add '.pipfile' suffix" \
            >> ${MANIFEST_PATH}   # append python deps to manifest
fi

## Write go deps
if [[ ! -z "$GO_SUM" ]]
then
    sed -e 's/\(\/go.mod\)\? h1:.*//; s/ /:/' "$GO_SUM" | sort | uniq \
        >> ${MANIFEST_PATH}   # append python deps to manifest
fi

## Add prefix to all lines.
sed -i -e "s|^|${PREFIX}|" ${MANIFEST_PATH}

## Write APP_VERSION if provided to :VERSION: placeholder
sed -i -e "s/:VERSION:/:${APP_VERSION:-latest}:/" ${MANIFEST_PATH}
