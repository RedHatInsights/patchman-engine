#!/bin/env python3
import os
from kafka import KafkaProducer

producer = KafkaProducer(bootstrap_servers=os.getenv("LISTENER_KAFKA_ADDRESS", None))
for i in range(1, int(os.getenv("BENCHMARK_MESSAGES", "30")) + 1):
    print(f"Sending message {i}")
    data = open(f"./data/body/{i}.json", 'rb').read()
    future = producer.send(os.getenv("LISTENER_KAFKA_TOPIC", None), data)
    future.get(timeout=60)

producer.flush(timeout=10)
