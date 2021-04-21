#!/bin/bash

rc=0

# Check consistent testing (centos) and production (redhat) dockerfiles.
for dockerfile in "Dockerfile" "database/Dockerfile"
do
    if [ ! -f "$dockerfile.centos" ]; then
        echo "Dockerfile '$dockerfile.centos' doesn't exist" >&2
        rc=$(($rc+1))
    fi
    for suffix in "rhel7" "rhel8"
    do
      if [ -f "$dockerfile.$suffix" ]; then
        sed \
            -e "s|centos/postgresql-12-centos8|registry.redhat.io/rhel8/postgresql-12|" \
            -e "s|centos:8|registry.access.redhat.com/ubi8|" \
            -e "s|RUN rpm --import /etc/pki/rpm-gpg/RPM-GPG-KEY-centosofficial||" \
            "$dockerfile.centos" | diff "${dockerfile}.$suffix" -
        diff_rc=$?
      if [ $diff_rc -gt 0 ]; then
        echo "$dockerfile and $dockerfile.$suffix are too different!"
      else
        echo "$dockerfile and $dockerfile.$suffix are OK"
      fi
      rc=$(($rc+$diff_rc))
      continue
    fi
    done
done
echo ""

exit $rc
