#!/bin/sh

#set -e
set -x

DIR=$(dirname $0)

cd $DIR

#KAFKA_HOME=$DIR/kafka
#
#echo "" >> $KAFKA_HOME/config/server.properties
#
## Set the external host and port
#if [ ! -z "$ADVERTISED_HOST" ]; then
#    echo "advertised host: $ADVERTISED_HOST"
#    if grep -q "^advertised.host.name" $KAFKA_HOME/config/server.properties; then
#        sed -r -i "s/#(advertised.host.name)=(.*)/\1=$ADVERTISED_HOST/g" $KAFKA_HOME/config/server.properties
#    else
#        echo "advertised.host.name=$ADVERTISED_HOST" >> $KAFKA_HOME/config/server.properties
#    fi
#fi
#if [ ! -z "$ADVERTISED_PORT" ]; then
#    echo "advertised port: $ADVERTISED_PORT"
#    if grep -q "^advertised.port" $KAFKA_HOME/config/server.properties; then
#        sed -r -i "s/#(advertised.port)=(.*)/\1=$ADVERTISED_PORT/g" $KAFKA_HOME/config/server.properties
#    else
#        echo "advertised.port=$ADVERTISED_PORT" >> $KAFKA_HOME/config/server.properties
#    fi
#fi


# run zookeeper
./kafka/bin/zookeeper-server-start.sh -daemon kafka/config/zookeeper.properties

# run kafka
./kafka/bin/kafka-server-start.sh -daemon kafka/config/server.properties

# wait until kafka starts
>&2 echo "Checking if Kafka server is up"
until ./kafka/bin/kafka-topics.sh --list --zookeeper localhost:2181 &> /dev/null; do
  >&2 echo "Kafka server is unavailable - sleeping"
  sleep 1
done

# create topics with multiple partitions for scaling
for topic in host.packages host.test
do
    ./kafka/bin/kafka-topics.sh --create --topic $topic --partitions 1 --zookeeper localhost:2181 --replication-factor 1
done

# generate kafka request messages
./generate_requests.py

# send messages to kafka
./send-kafka-requests.sh

# run upload mock
exec ./wait-for-services.sh sleep 5000000
