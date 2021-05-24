#!/bin/sh

# ./scripts/generate_rpm_list.sh
# Example:
# ./scripts/generate_rpm_list.sh "exclude1|exclude2"

EXCLUDE="\(none\)|kernel-"
if [ -n "$1" ] ; then
    EXCLUDE="$EXCLUDE|$1"
fi

rpm -qa --qf='%{sourcerpm}\n' | grep -vE "^($EXCLUDE)" | sort -u | sed 's/\.src\.rpm$//'
