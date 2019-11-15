#!/usr/bin/env bash

set -x
trap ctrl_c INT

# Load up common env variables
set -o allexport
source conf/common.env
set +o allexport

COUNT=${BENCHMARK_MESSAGES}

function ctrl_c() {
  kill -9 $BG_PID
  kill -9 $go_PID
  kill -9 $python_PID
  kill -9 $rust_PID
  exit  1
}

function start_collection(){
  ./dev/collect-stats.sh out/usages.csv &
  BG_PID=$!
}

function stop_collection() {
  kill -9 $BG_PID
}

function send_message_batch() {
  echo "Sending a new batch of requests"
  docker exec -it platform bash -c "./send_kafka_requests.py"
}

function perform_benchmark() {
  TYPE=$1
  PORT=$2

  echo "Building the $TYPE container"
  docker-compose build $TYPE
  echo "Starting the $TYPE container benchmark"
  docker-compose up --build $TYPE  | tee out/$TYPE.log &

  sleep 10
  until curl -XGET -u admin:passwd --fail http://localhost:$PORT/hosts/$COUNT > out/$TYPE-$COUNT.json 2> /dev/null;
  do
    sleep 1
  done

  sleep 5

  echo "Retrieved last item, all must be processed"
  echo "$(curl -XGET -u admin:passwd  http://localhost:$PORT/hosts/1)" > out/$TYPE-1.json

  docker-compose stop $TYPE

}

docker-compose down

docker-compose up --build -d platform db

echo "Waiting for db to come up"
docker exec -it platform bash -c "./wait-for-services.sh true"


mkdir -p out
echo "Starting the stats collection"


start_collection
perform_benchmark go 8080
stop_collection

send_message_batch


start_collection
perform_benchmark rust 8082
stop_collection


send_message_batch


start_collection
perform_benchmark python 8081
stop_collection


ctrl_c