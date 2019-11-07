import db

import hashlib

from playhouse.shortcuts import model_to_dict, dict_to_model

# def all_hosts():
#     host = db.Host.select(
#         db.Host.id,
#         db.Host.request,
#         db.Host.checksum)
#
#     return [host for host in hosts.dicts()]

def get_host(id):
    host = db.Host.select(
        db.Host.id,
        db.Host.request,
        db.Host.checksum
    ).where(db.Host.id == id).get()

    if not host:
        return None, 404

    host = model_to_dict(host)

    request = host['request']
    checksum = hashlib.sha256(request).hexdigest()

    if checksum != request['checksum']:
        return {"err": "Invalid checksum"}, 206

    return host

