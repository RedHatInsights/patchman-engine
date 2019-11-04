"""
ThreadPoolExecutor wrapper to limit number of submited items.
"""

from concurrent.futures import ThreadPoolExecutor
from threading import BoundedSemaphore


class BoundedExecutor:
    """
    BoundedExecutor behaves as a ThreadPoolExecutor which will block on
    calls to submit() once the limit given as "bound" work items are queued for
    execution.
    :param max_queue_size: Integer - the maximum number of items in the work queue
    :param max_workers: Integer - the size of the thread pool
    """
    def __init__(self, max_queue_size, max_workers=None):
        self.semaphore = BoundedSemaphore(max_queue_size)
        self.executor = ThreadPoolExecutor(max_workers=max_workers)

    def submit(self, func, *args, **kwargs):
        """blocking submit method"""
        self.semaphore.acquire()
        try:
            future = self.executor.submit(func, *args, **kwargs)
        except:  # noqa: E722
            self.semaphore.release()
            raise
        else:
            future.add_done_callback(lambda x: self.semaphore.release())
            return future

    def shutdown(self, wait=True):
        """pass shutdown to executor"""
        self.executor.shutdown(wait=wait)
