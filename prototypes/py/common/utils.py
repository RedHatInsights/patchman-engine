"""
Various utility functions.
"""

import os
from time import sleep
from datetime import datetime

from prometheus_client import Counter
import requests

from common.logging import get_logger

VMAAS_REQUEST_RETRIES = int(os.getenv("VMAAS_REQUEST_RETRIES", "3"))

LOGGER = get_logger(__name__)

# prometheus probes
# counts
VMAAS_RETURN_ERR = Counter('ve_evaluator_vmaas_return_errors', 'Number of non-200 RCs from VMaaS')
VMAAS_CNX_ERR = Counter('ve_evaluator_vmaas_cnx_errors', 'Number of connection-errors from VMaaS')


def vmaas_post_request(endpoint, data_json, session=None):
    """Sends request to VMAAS"""
    headers = {'Content-type': 'application/json',
               'Accept': 'application/json'}
    for _ in range(VMAAS_REQUEST_RETRIES):
        try:
            if session:
                response = session.post(endpoint, json=data_json, headers=headers)
            else:
                response = requests.post(endpoint, json=data_json, headers=headers)
            if response.status_code == 200:
                return response.json()
            VMAAS_RETURN_ERR.inc()
            LOGGER.error("Error during request to VMaaS endpoint %s: HTTP %s, %s",
                         endpoint, response.status_code, response.text)
            LOGGER.debug("JSON: %s", str(data_json))
        except requests.exceptions.RequestException:
            VMAAS_CNX_ERR.inc()
            LOGGER.exception("Error calling VMAAS: ")
        sleep(0.1)
    return None


def on_thread_done(future):
    """Callback to call after ThreadPoolExecutor worker finishes."""
    try:
        future.result()
    except Exception:  # pylint: disable=broad-except
        LOGGER.exception("Future %s hit exception: ", future)


def str_or_none(value):
    """Return string or None i value not exist"""
    return str(value) if value else None


def format_datetime(datetime_obj):
    """Convert datetime format to string ISO format"""
    if isinstance(datetime_obj, datetime):
        return datetime_obj.isoformat()
    return str(datetime_obj) if datetime_obj else None
