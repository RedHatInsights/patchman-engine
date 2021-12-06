#!/usr/bin/bash

# Wait for closing of all "listener", "evaluator" and "vmaas_sync" database sessions.

while :
do
  SESSIONS=$(psql -t -c "SELECT usename, substring(query for 50) FROM pg_stat_activity WHERE usename IN ('evaluator', 'listener', 'vmaas_sync') LIMIT 30;")
  if [[ $SESSIONS == "" ]]; then
    echo "No 'listener', 'evaluator', 'vmaas_sync' sessions found"
    break
  else
    echo "Sessions found:"
    echo $SESSIONS
    sleep 1
  fi
done
