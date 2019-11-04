"""
Module handling page separation
"""

import requests

from common.logging import get_logger
from common import utils

LOGGER = get_logger(__name__)


def paging(endpoint, request_json) -> tuple:
    """Split input list into pages"""
    result_pages = {}
    session = requests.Session()
    success = True
    while True:
        r_json = utils.vmaas_post_request(endpoint, request_json, session)
        if r_json is None:
            LOGGER.info("Downloading ERROR.")
            success = False
            break
        response_to_list = [(k, v) for k, v in r_json.items()]
        data_index = response_to_list[0][0]
        data = response_to_list[0][1]
        result_pages.setdefault(data_index, {}).update(data)
        LOGGER.info("Downloading CVE/REPOs metadata (page: %s, page_size: %s, pages: %s)",
                    request_json['page'], r_json["page_size"], r_json['pages'])
        if request_json['page'] >= r_json['pages']:
            break
        request_json['page'] += 1
    session.close()

    if success:
        result_pages.update({"page": r_json["page"], "page_size": r_json["page_size"], "pages": r_json["pages"]})
    return (success, result_pages)
