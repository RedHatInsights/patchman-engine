#!/usr/bin/env bash

set -e

DATA_FILE=$1
timestamp() {
  date -u +"%T"
}

while true; do
  STATS=$(docker stats --no-stream | tail -n +2)
  IFS=$'\n'
  for l in $STATS; do
    ID=$(echo $l | tr -s ' ' | cut -d' ' -f1)
    CPU=$(echo $l | tr -s ' ' | cut -d' ' -f2)
    MEM=$(echo $l | tr -s ' ' | cut -d' ' -f3)
    NAME=$(docker ps --format '{{.ID}} {{.Names}}' | grep $ID | tr -s 't' | cut -d' ' -f2)

    #echo "NAME:" $NAME $'\t\t' "CPU: " $CPU $'\t' "MEM: " $MEM
    echo $(timestamp) "," $NAME "," $CPU "," $MEM >> $DATA_FILE
  done
done
