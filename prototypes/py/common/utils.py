"""
Various utility functions.
"""

import os
from time import sleep
from datetime import datetime
import re

from common.logging import get_logger
LOGGER = get_logger(__name__)

NEVRA_RE = re.compile(r'(.*)-(([0-9]+):)?([^-]+)-([^-]+)\.([a-z0-9_]+)')
def split_packagename(filename):
    """
    Split rpm name (incl. epoch) to NEVRA components.

    Return a name, epoch, version, release, arch, e.g.::
        foo-1.0-1.i386.rpm returns foo, 0, 1.0, 1, i386
        bar-1:9-123a.ia64.rpm returns bar, 1, 9, 123a, ia64
    """

    if filename[-4:] == '.rpm':
        filename = filename[:-4]

    match = NEVRA_RE.match(filename)
    if not match:
        return '', '', '', '', ''

    name, _, epoch, version, release, arch = match.groups()
    if epoch is None:
        epoch = '0'
    return name, epoch, version, release, arch

def join_packagename(name, epoch, version, release, arch):
    """
    Build a package name from the separate NEVRA parts
    """
    if name and epoch and version and release and arch:
        try:
            epoch = ("%s:" % epoch) if int(epoch) else ''
        except Exception as _: # pylint: disable=broad-except
            epoch = ''
        return "%s-%s%s-%s.%s" % (name, epoch, version, release, arch)

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
