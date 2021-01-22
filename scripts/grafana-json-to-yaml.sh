#!/bin/bash

PREFIX="apiVersion: v1\n\
data:\n\
  grafana.json: |"
POSTFIX="kind: ConfigMap\n\
metadata:\n\
  name: grafana-dashboard-insights-patchman-engine-general\n\
  labels:\n\
    grafana_dashboard: \"true\"\n\
  annotations:\n\
    grafana-folder: /grafana-dashboard-definitions/Insights"

sed "1 i $PREFIX
     $ a $POSTFIX
     s/^/    /
     " $1

exit 0
{
    echo "$PREFIX"
    cat $1
    echo "$POSTFIX"
} 
