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
if [[ ! -z $PYTHON ]]
then
    $PYTHON -m pip freeze | sort > /tmp/pipdeps
    sed -i -e 's/^/'$PYTHON'-/' /tmp/pipdeps  # add 'python' prefix
    sed -i -e 's/==/-/' /tmp/pipdeps       # replace '==' with '-'
    sed -i -e 's/$/.pipfile/' /tmp/pipdeps # add '.pipfile' suffix
    cat /tmp/pipdeps >> ${MANIFEST_PATH}   # append python deps to manifest
fi

## Write go deps
if [[ ! -z $GO_SUM ]]
then
    cat $GO_SUM | sed -e 's/\(\/go.mod\)\? h1:.*//' | sed 's/ /:/' | sort | uniq > /tmp/godeps
    cat /tmp/godeps >> ${MANIFEST_PATH}   # append python deps to manifest
fi

## Add prefix to all lines.
sed -i -e 's/^/'${PREFIX}'/' ${MANIFEST_PATH}

## Write APP_VERSION if provided to :VERSION: placeholder
if [[ ! -z $APP_VERSION ]]
then
    sed -i -e 's/:VERSION:/:'$APP_VERSION':/' ${MANIFEST_PATH}
else
    sed -i -e 's/:VERSION:/:latest:/' ${MANIFEST_PATH}
fi
