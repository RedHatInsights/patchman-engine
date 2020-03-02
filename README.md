[![Build Status](https://travis-ci.org/RedHatInsights/patchman-engine.svg?branch=master)](https://travis-ci.org/RedHatInsights/patchman-engine)
[![Code Coverage](https://codecov.io/gh/RedHatInsights/patchman-engine/branch/master/graph/badge.svg)](https://codecov.io/gh/RedHatInsights/patchman-engine)

# patchman-engine
System Patch Manager is one application for [cloud.redhat.com](cloud.redhat.com). See [architecture](docs/md/architecture.md) and [database](docs/md/database.md) for details.

### Monitoring
Each application component (except database) exposes metrics for [Prometheus](https://prometheus.io/)
on `/metrics` endpoint (see [docker-compose.yml](docker-compose.yml) for ports). Runtime logs can be send to Amazon
CloudWatch setting proper environment variables (see [awscloudwatch.go](base/utils/awscloudwatch.go)).

## Deploying
This project can be deployed either locally or in the cloud using openshift.

### Local deployment
Uses `podman-compose` to deploy the individual project components and supporting containers, which simulate the CMSfR platform and database respectively into local container instance:
~~~bash
podman-compose up --build # Build images if needed and start containers
podman-compose down       # Stop and remove containers
~~~

## Test local-running app
When podman compose is running, test app using dev shell scripts:
~~~bash
cd dev/scripts
./systems_list.sh         # show systems
./advisories_list.sh      # show advisories
./platform_sync.sh        # trigger vmaas_sync to sync (using vmaas mock)
./platform_upload.sh      # simulate archive upload to trigger listener and evaluator_upload
~~~

#### VMaaS
This project uses [VMaaS](https://github.com/RedHatInsights/vmaas) for retrieving information about advisories, and resolving which advisories can be applied to whic systems.
For local development, you need to clone VMaaS, and deploy it alongside this project.

## (Re)generate API docs
~~~bash
go get -u github.com/swaggo/swag/cmd/swag # download binary to generate, do it first time only
./scripts/generate_docs.sh
~~~

Test using Swagger, open <http://localhost:8080/openapi/index.html>.

## Run tests
~~~bash
podman-compose -f docker-compose.test.yml up --build --abort-on-container-exit
~~~

## Run vmaas_sync "sync" manually
There is a private API accessible only from inside of `vmaas_sync` container:
~~~
podman exec -it patchman-engine_vmaas_sync_1 ./sync.sh
~~~
