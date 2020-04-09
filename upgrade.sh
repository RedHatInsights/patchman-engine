#!/bin/bash

set -euo pipefail

buildconfigs="patchman-engine-app"
deploymentconfigs="
patchman-engine-database-admin
patchman-engine-evaluator-recalc
patchman-engine-evaluator-upload
patchman-engine-listener
patchman-engine-manager
patchman-engine-vmaas-sync"

function upgrade-builds() {
  VERSION=$1
  TAG=${2:-VERSION}
  PATCH=$(envsubst <<EOF
{
  "spec": {
    "output": {
      "to": {
        "name": "patchman-engine-app:$VERSION"
      }
    },
    "source": {
      "git": {
        "ref": "$TAG"
      }
    }
  }
}
EOF
)
  oc patch bc patchman-engine-app -p "$PATCH"
  oc start-build patchman-engine-app -w
}

function upgrade-services() {
  VERSION=$1
  for DC in $deploymentconfigs; do
    PATCH=$(envsubst <<EOF
{
  "spec": {
    "triggers": [
      {"type": "ConfigChange"},
      {
        "type": "ImageChange",
        "imageChangeParams": {
          "automatic": true,
          "containerNames": ["$DC"],
          "from": {
            "name": "patchman-engine-app:$VERSION"
          }
        }
      }
    ]
  }
}
EOF
)
    oc patch dc $DC -p "$PATCH"
  done
}

function help() {
  cat <<EOF

Commands:
  upgrade-builds VERSION [GIT_TAG]
    - Upgrades buildconfigs to take GIT_TAG and output images with VERSION tag,
     by default, the GIT_TAG is set to VERSION
  upgrade-services VERSION
    - Upgrades DeploymentConfigs to use images with VERSION tag

EOF
}

$@
