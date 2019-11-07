import db
import json

import hashlib

from playhouse.shortcuts import model_to_dict, dict_to_model

def all_hosts():
    hosts = db.Host.select(
        db.Host.id,
        db.Host.request,
        db.Host.checksum)
    return [host for host in hosts.dicts()]

def get_host(id):
    host = db.Host.get_or_none(db.Host.id == id)

    if not host:
        return None, 404

    host = model_to_dict(host)

    request = host['request']
    checksum = hashlib.sha256(request.encode('utf-8')).hexdigest()
    request = json.loads(request)

    if checksum != host['checksum']:
        return {"err": "Invalid checksum"}, 206

    return host

