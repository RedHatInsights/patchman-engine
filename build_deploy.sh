#!/bin/bash

set -exv

# no space left on jenkins
export TMPDIR=/var/lib/jenkins

IMAGE="quay.io/cloudservices/patchman-engine-app"
IMAGE_TAG=$(git rev-parse --short=7 HEAD)
IMAGE_VERSION=$(git tag --points-at $IMAGE_TAG)
SECURITY_COMPLIANCE_TAG="sc-$(date +%Y%m%d)-$(git rev-parse --short=7 HEAD)"

if [[ -z "$QUAY_USER" || -z "$QUAY_TOKEN" ]]; then
    echo "QUAY_USER and QUAY_TOKEN must be set"
    exit 1
fi

if [[ -z "$RH_REGISTRY_USER" || -z "$RH_REGISTRY_TOKEN" ]]; then
    echo "RH_REGISTRY_USER and RH_REGISTRY_TOKEN  must be set"
    exit 1
fi

# Create tmp dir to store data in during job run (do NOT store in $WORKSPACE)
export TMP_JOB_DIR=$(mktemp -d -p "$HOME" -t "jenkins-${JOB_NAME}-${BUILD_NUMBER}-XXXXXX")
echo "job tmp dir location: $TMP_JOB_DIR"

function job_cleanup() {
    echo "cleaning up job tmp dir: $TMP_JOB_DIR"
    rm -fr $TMP_JOB_DIR
}

trap job_cleanup EXIT ERR SIGINT SIGTERM

AUTH_CONF_DIR="$TMP_JOB_DIR/.podman"
mkdir -p $AUTH_CONF_DIR
export REGISTRY_AUTH_FILE="$AUTH_CONF_DIR/auth.json"
podman login -u="$QUAY_USER" -p="$QUAY_TOKEN" quay.io
podman login -u="$RH_REGISTRY_USER" -p="$RH_REGISTRY_TOKEN" registry.redhat.io
podman build -f Dockerfile -t "${IMAGE}:${IMAGE_TAG}" .

if [[ "$GIT_BRANCH" == "origin/security-compliance" ]]; then
    podman tag "${IMAGE}:${IMAGE_TAG}" "${IMAGE}:${SECURITY_COMPLIANCE_TAG}"
    podman push "${IMAGE}:${SECURITY_COMPLIANCE_TAG}"
else
    podman push "${IMAGE}:${IMAGE_TAG}"
    podman tag "${IMAGE}:${IMAGE_TAG}" "${IMAGE}:latest"
    podman push "${IMAGE}:latest"
    if [[ -n "$IMAGE_VERSION" ]]; then
        podman tag "${IMAGE}:${IMAGE_TAG}" "${IMAGE}:${IMAGE_VERSION}"
        podman push "${IMAGE}:${IMAGE_VERSION}"
    fi
fi
