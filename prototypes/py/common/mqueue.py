"""Module with classes for work with Kafka Message queue."""

import asyncio
import json
import os

from aiokafka import AIOKafkaConsumer, AIOKafkaProducer
from kafka.errors import KafkaError

from common.logging import get_logger

LOGGER = get_logger(__name__)

BOOTSTRAP_SERVERS = "%s:%s" % (os.getenv("KAFKA_HOST", "127.0.0.1"),
                               os.getenv("KAFKA_PORT", "9092"))


GROUP_ID = os.getenv('KAFKA_GROUP_ID', 'spm')

RETRY_INTERVAL = int(os.getenv('RETRY_INTERVAL', '5'))


class MQClient:
    """Message queue client wrapper around AIOKafka"""

    def __init__(self, client, name):
        self.client = client
        self.name = name
        self.connected = False
        self.connect_lock = asyncio.Lock()

    async def start(self):
        """Starts the kafka consumer/producer"""
        await self.connect_lock.acquire()
        while not self.connected:
            try:
                LOGGER.info("Attempting to connect %s client.", self.name)
                await self.client.start()
                LOGGER.info("%s client connected successfully.", self.name)
                self.connected = True
            except KafkaError:
                LOGGER.exception("Failed to connect %s client, retrying in %d seconds.", self.name, RETRY_INTERVAL)
                await asyncio.sleep(RETRY_INTERVAL)
        self.connect_lock.release()

    async def stop(self):
        """Stops the kafka consumer/producer"""
        try:
            LOGGER.info("Attempting to stop %s client.", self.name)
            await self.client.stop()
            LOGGER.info("%s client stopped successfully.", self.name)
            self.connected = False
        except KafkaError:
            LOGGER.exception("Failed to stop %s client.", self.name)


class MQReader(MQClient):
    """Wrapper around AIOKafka consumer"""

    def __init__(self, topic, group_id=GROUP_ID, bootstrap_servers=BOOTSTRAP_SERVERS, **kwargs):
        self.loop = asyncio.get_event_loop()
        if isinstance(topic, str):
            topic = [topic]
        consumer = AIOKafkaConsumer(*topic, loop=self.loop, bootstrap_servers=bootstrap_servers, group_id=group_id,
                                    auto_offset_reset='latest', session_timeout_ms=30000, **kwargs)
        super(MQReader, self).__init__(consumer, 'AIOKafka Consumer')

    async def consume(self, func):
        """Consmes messages comming from Kafka"""
        await self.start()
        try:
            async for msg in self.client:
                func(msg)
        except KafkaError:
            self.connected = False

    def listen(self, func):
        """Starts the listener"""
        self.loop.run_until_complete(self.consume(func))


class MQWriter(MQClient):
    """Wrapper around AIOKafka producer"""

    def __init__(self, topic, bootstrap_servers=BOOTSTRAP_SERVERS, **kwargs):
        self.loop = asyncio.get_event_loop()
        producer = AIOKafkaProducer(loop=self.loop, bootstrap_servers=bootstrap_servers, **kwargs)
        self.topic = topic
        super(MQWriter, self).__init__(producer, 'AIOKafka Producer')

    async def send_one(self, msg):
        """Logic around message sending"""
        await self.start()
        try:
            data = bytes(json.dumps(msg).encode('utf-8'))
            res = await self.client.send_and_wait(self.topic, data)
            LOGGER.debug(res)
        except KafkaError:
            self.connected = False

    async def send_many(self, msg_list):
        """Send list of messages"""
        await self.start()
        try:
            for msg in msg_list:
                data = bytes(json.dumps(msg).encode('utf-8'))
                res = await self.client.send_and_wait(self.topic, data)
                LOGGER.debug(res)
        except KafkaError:
            self.connected = False

    def send(self, msg, loop=None):
        """Sends a message"""
        return asyncio.ensure_future(self.send_one(msg), loop=loop)

    def send_list(self, msgs, loop=None):
        """Sends list os messages"""
        return asyncio.ensure_future(self.send_many(msgs), loop=loop)
