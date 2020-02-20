#!/bin/sh

# GIT_TOKEN=<GIT_TOKEN> ./scripts/manifest/push.sh <GIT_REPO> <GIT_BRANCH> <SOURCE_FILE_PATH> <GIT_FILE_PATH>
# Example:
# GIT_TOKEN=mygithubtoken ./scripts/manifest/push.sh RedHatInsights/manifests master /manifest.txt patchman-engine/patchman-engine.txt

GIT_REPO=$1
GIT_BRANCH=$2
SOURCE_FILE_PATH=$3
GIT_FILE_PATH=$4

API_ENDPOINT="https://api.github.com/repos/$GIT_REPO/contents/$GIT_FILE_PATH"

if [[ ! -z $GIT_TOKEN ]]
then
    retry=0
    until [ $retry -ge 5 ]
    do
        curl -H "Authorization: token $GIT_TOKEN" -X GET $API_ENDPOINT?ref=$GIT_BRANCH | grep -oP '(?<="content": ").*(?=")' | sed 's/\\n//g' | base64 -d | diff $SOURCE_FILE_PATH -
        diff_rc=$?
        if [ $diff_rc -eq 0 ]
        then
            echo "Remote manifest is already up to date!"
            break
        fi
        # fetch remote file sha (if exists)
        remote_file_sha=$(curl -H "Authorization: token $GIT_TOKEN" -X GET $API_ENDPOINT?ref=$GIT_BRANCH | grep -oP '(?<="sha": ").*(?=")')
        # insert or update file
        echo "{\"message\": \"Updating $GIT_FILE_PATH\", \"branch\": \"$GIT_BRANCH\", \"sha\": \"$remote_file_sha\", \"content\": \"$(base64 -w 0 $SOURCE_FILE_PATH)\"}" > /tmp/commit_payload.json
        new_commit_sha=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: token $GIT_TOKEN" -X PUT -d "@/tmp/commit_payload.json" $API_ENDPOINT)
        # curl return 200 or 201 => success
        if [[ $new_commit_sha == "200" ]] || [[ $new_commit_sha == "201" ]]
        then
            break
        fi
        retry=$((retry+1))
        echo "Update failed, trying again after 1 second..."
        sleep 1
    done
else
    echo "GIT_TOKEN is not set, not pushing anything."
fi
