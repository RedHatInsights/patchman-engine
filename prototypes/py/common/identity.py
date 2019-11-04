"""
3scale authentication functions.
"""

import base64
from distutils.util import strtobool  # pylint: disable=import-error, no-name-in-module
import json
import os

from common.logging import get_logger

SKIP_ENTITLEMENT_CHECK = strtobool(os.getenv('SKIP_ENTITLEMENT_CHECK', 'FALSE'))

LOGGER = get_logger(__name__)


def get_identity(x_rh_identity: str) -> dict:
    """Get identity from given b64 string."""
    try:
        decoded_value = base64.b64decode(x_rh_identity).decode("utf-8")
    except Exception:  # pylint: disable=broad-except
        LOGGER.warning("Error decoding b64 string: %s", x_rh_identity)
        decoded_value = ""
    else:
        LOGGER.debug("Identity decoded: %s", decoded_value)
    try:
        identity = json.loads(decoded_value)
    except json.decoder.JSONDecodeError:
        LOGGER.warning("Error parsing JSON identity: %s", decoded_value)
        identity = None
    return identity


def is_entitled_smart_management(identity: dict, allow_missing_section: bool = False) -> bool:
    """Check if given identity has smart_management entitlement"""
    if SKIP_ENTITLEMENT_CHECK:
        return True
    if allow_missing_section and "entitlements" not in identity:
        # FIXME: remove this when all incoming identities have entitlements section
        LOGGER.info("Entitlements section is missing in identity, marking as entitled.")
        return True
    return identity.get("entitlements", {}).get("smart_management", {}).get("is_entitled", False)
