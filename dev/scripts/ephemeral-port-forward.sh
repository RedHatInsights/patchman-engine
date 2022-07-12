#!/bin/bash

#PODS="$(oc get services -o name)"

fwd() {
    service=$1
    ports=$2

    oc port-forward "$service" $ports >/dev/null &
    echo $service forwarded to $ports
}

fwd service/ingress-service 8000:8000
fwd service/patchman-db 5433:5432
fwd service/patchman-manager 8080:8000

wait
