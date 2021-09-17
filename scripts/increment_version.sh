#!/bin/bash
# Based on gist: https://gist.github.com/siddharthkrish/32072e6f97d7743b1a7c47d76d2cb06c#file-version-sh
# Usage: ./scripts/increment_version.sh v1.2.3 /major
# v2.0.0
#        ./scripts/increment_version.sh v1.2.3 /minor
# v1.3.0
#        ./scripts/increment_version.sh v1.2.3
# v1.2.4

version="$1"
RELEASE_TYPE=$2 # /major, /minor, /patch (default)

major=0
minor=0
build=0

# break down the version number into it's components
regex="([0-9]+).([0-9]+).([0-9]+)"
if [[ $version =~ $regex ]]; then
  major="${BASH_REMATCH[1]}"
  minor="${BASH_REMATCH[2]}"
  build="${BASH_REMATCH[3]}"
fi

if [[ "${RELEASE_TYPE}" == "/major" ]]; then
  ((major++))
  minor=0
  build=0
elif [[ "${RELEASE_TYPE}" == "/minor" ]]; then
  ((minor++))
  build=0
else
  ((build++))
fi

# echo the new version number
echo "v${major}.${minor}.${build}"
