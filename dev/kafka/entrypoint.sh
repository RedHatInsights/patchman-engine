#!/bin/sh

#wait until zookeeper is ready
IFS=: read zooserver zooport <<<"$KAFKA_ZOOKEEPER_CONNECT"
while : ; do
    STATUS=$(nc $zooserver $zooport <<<"ruok" 2>/dev/null)
    if [[ $STATUS == 'imok' ]] ; then
        break
    fi
    >&2 echo "Waiting until zookeeper is running"
    sleep 1
done

/app/setup.sh 2>&1 | grep -v '^WARNING: Due to limitations in metric names' &

exec /etc/confluent/docker/run 2>&1 \
    | grep -v -E ' (TRACE|DEBUG|INFO) '
