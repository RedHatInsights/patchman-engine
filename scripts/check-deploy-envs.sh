#!/bin/sh

YAML=deploy/clowdapp.yaml 

ENV_VARS=$(grep '\${.*}' $YAML | sed 's/.*\${\+\([^}]*\)}\+.*/\1/')

for i in $ENV_VARS ; do
        if awk "/parameters:/ {params=1} params && /$i/ { exit 1;}" $YAML ; then
            >&2 echo "Value of $i is not defined in $YAML"
            ERROR=1
        fi
done

[[ -z $ERROR ]] || exit 1
