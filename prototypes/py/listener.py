from common.mqueue import *
from common.utils import split_packagename, join_packagename

from common.bounded_executor import BoundedExecutor
from playhouse.shortcuts import model_to_dict, dict_to_model


import db
import json
import hashlib
import asyncio
import os
import time

LOGGER = get_logger(__name__)
LISTENER_TOPIC = os.getenv("LISTENER_KAFKA_TOPIC", "host.packages")
BUFFER_SIZE = int(os.getenv("LISTENER_BUFFER_SIZE", "10"))

MAX_QUEUE_SIZE = BUFFER_SIZE
WORKER_THREADS = 8

class Benchmark:

    def __init__(self):
        self.benchmark_msgs = int(os.getenv("BENCHMARK_MESSAGES", '30'))
        self.msgs = []
        self.start_time = None

    def save_host(self, msg):
        """Saves a single host"""

        # Lazy initialize the start_time
        if not self.start_time:
            self.start_time = time.perf_counter()

        self.msgs.append(msg)
        #host = dict_to_model(db.Host, data=msg)
        LOGGER.info(db.Host.insert(msg).execute())

        if len(self.msgs) == self.benchmark_msgs:
            finish_time = time.perf_counter()
            LOGGER.info(f"Saved {self.benchmark_msgs} msgs,  write/s:{float(self.benchmark_msgs)/(finish_time - self.start_time)}")
            self.msgs.clear()
            self.start_time = None


def process_msg(msg, bench):
    try:
        data = json.loads(msg.value)

        host_id = data['id']
        host_arch = data['arch']
        res_packages = []

        for p in data['packages']:
            name, epoch, version, release, arch = split_packagename(p)
            if arch == host_arch:
                res_packages.append(join_packagename(name, epoch, version, release, arch))

        req = {
            'package_list': res_packages,
        }

        req = json.dumps(req).encode('utf-8')
        checksum = hashlib.sha256(req).hexdigest()

        bench.save_host({
            'id': host_id,
            'request': req,
            'checksum': checksum
        })

    except Exception as e:
        LOGGER.error(e)

    pass


def main():
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)

    executor = BoundedExecutor(MAX_QUEUE_SIZE, max_workers=WORKER_THREADS)

    bench = Benchmark()

    def process(msg):
        executor.submit(process_msg, msg, bench)

    reader = MQReader(LISTENER_TOPIC,)
    reader.listen(process)

    loop.run_forever()
    LOGGER.info("Shutting down.")
