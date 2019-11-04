"""
Common logging functionality to be used for multiple apps
"""

from distutils.util import strtobool  # pylint: disable=import-error, no-name-in-module
import logging
import os
import watchtower
from boto3.session import Session
from botocore.exceptions import ClientError


class OneLineExceptionFormatter(logging.Formatter):
    """
    Formatter used to insure each log-entry is one line
    (insures one entry-per-log for some logging environments that divide on newline)
    """

    # pylint: disable=arguments-differ
    def formatException(self, exc_info):
        """
        Make sure exception-tracebacks end up on a single line.
        """
        result = super(OneLineExceptionFormatter, self).formatException(exc_info)
        return repr(result)

    def format(self, record):
        """
        Convert newlines in each record to |
        """
        fmt_str = super(OneLineExceptionFormatter, self).format(record)
        if record.exc_text:
            fmt_str = fmt_str.replace('\n', '') + '|'
        return fmt_str


def setup_cw_logging(main_logger):
    """Setup CloudWatch logging"""
    logger = get_logger(__name__)
    if not strtobool(os.getenv('CW_ENABLED', 'FALSE')):
        logger.info('CloudWatch logging disabled')
        return
    key_id = os.environ.get('CW_AWS_ACCESS_KEY_ID')
    secret = os.environ.get('CW_AWS_SECRET_ACCESS_KEY')
    if not (key_id and secret):
        logger.info('CloudWatch logging disabled due to missing access key')
        return

    try:
        with open('/var/run/secrets/kubernetes.io/serviceaccount/namespace', 'r') as namespace_file:
            namespace = namespace_file.read()  # pragma: no cover
    except Exception:  # pylint: disable=broad-except
        namespace = 'vulnerability-engine-development'

    session = Session(
        aws_access_key_id=key_id,
        aws_secret_access_key=secret,
        region_name=os.environ.get('AWS_REGION', 'us-east-1'),
    )

    try:
        handler = watchtower.CloudWatchLogHandler(
            boto3_session=session,
            log_group=os.environ.get('CW_LOG_GROUP', 'platform-dev'),
            stream_name=namespace
        )
    except ClientError:
        logger.exception("Unable to enable CloudWatch logging: ")
    else:  # pragma: no cover
        main_logger.addHandler(handler)
        logger.info('CloudWatch logging ENABLED!')


def init_logging(num_servers=1):
    """Setup root logger handler."""
    logger = logging.getLogger()
    log_type = os.getenv('LOGGING_TYPE', "DEVEL")
    log_fmt = "%(asctime)s:%(levelname)s:%(name)s:%(message)s"
    if log_type != "OPENSHIFT":
        uuid = os.uname().nodename
        if num_servers > 1:
            uuid += ":%d" % os.getpid()
        log_fmt = "%s %s" % (uuid, log_fmt)
    level = os.getenv('LOGGING_LEVEL_LIBS', "WARNING")
    logger.setLevel(getattr(logging, level, logging.WARNING))
    if not logger.handlers:
        handler = logging.StreamHandler()
        formatter = OneLineExceptionFormatter(log_fmt)
        handler.setFormatter(formatter)
        logger.addHandler(handler)
    setup_cw_logging(logger)


def get_logger(name):
    """
    Set logging level and return logger.
    Don't set custom logging level in root handler to not display debug messages from underlying libraries.
    """
    logger = logging.getLogger(name)
    level = os.getenv('LOGGING_LEVEL_APP', "INFO")
    logger.setLevel(getattr(logging, level, logging.INFO))
    return logger
