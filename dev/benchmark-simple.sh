#!/usr/bin/env bash

trap ctrl_c INT

COUNT=100

function ctrl_c() {
  kill -9 $BG_PID
  kill -9 $go_PID
  kill -9 $python_PID
  kill -9 $rust_PID
  exit  1
}

function perform_benchmark() {
  TYPE=$1
  PORT=$2

  echo "Building the $TYPE container"
  docker-compose build $TYPE
  echo "Starting the $TYPE container benchmark"
  docker-compose up --build $TYPE  | tee out/$TYPE.log &
  "${TYPE}_PID"=$!

  sleep 10
  until curl -XGET -u admin:passwd --fail http://localhost:$PORT/hosts/$COUNT > out/$TYPE-$COUNT.json 2> /dev/null;
  do
    sleep 1
  done

  echo "Retrieved last item, all must be processed"
  echo $(curl -XGET -u admin:passwd  http://localhost:$PORT/hosts/1) > out/$TYPE-1.json

  docker-compose stop $TYPE


  echo "Sending a new batch of requests"
  docker exec -it platform bash -c "./send_kafka_requests.py"
}



docker-compose down

docker-compose up --build -d platform db

echo "Waiting for db to come up"
docker exec -it platform bash -c "./wait-for-services.sh true"



mkdir -p out

echo "Starting the stats collection"
./dev/collect-stats.sh out/usages.csv &
BG_PID=$!



sleep 2

perform_benchmark go 8080
perform_benchmark python 8081

ctrl_c