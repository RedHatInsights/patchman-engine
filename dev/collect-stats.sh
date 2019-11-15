#!/usr/bin/env bash


DATA_FILE=$1
timestamp() {
  date -u +"%T"
}

while true; do
  STATS=$(docker stats --format "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" --no-stream)
  IFS=$'\n'
  for l in $STATS; do
    NAME=$(echo $l | cut -d$'\t' -f1)
    CPU=$(echo $l  | cut -d$'\t' -f2)
    MEM=$(echo $l  | cut -d$'\t' -f3 | cut -d'/' -f1 )
    echo $(timestamp) "," $NAME "," $CPU "," $MEM >> $DATA_FILE
  done
done
