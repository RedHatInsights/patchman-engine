[![Build Status](https://travis-ci.org/RedHatInsights/patchman-engine.svg?branch=master)](https://travis-ci.org/RedHatInsights/patchman-engine)
[![Code Coverage](https://codecov.io/gh/RedHatInsights/patchman-engine/branch/master/graph/badge.svg)](https://codecov.io/gh/RedHatInsights/patchman-engine)

# patchman-engine
System Patch Manager is one of the applications for [cloud.redhat.com](cloud.redhat.com). This application allows to display and manage available patches for account systems. This repo stores sources for the backend part of the application providing REST API to the frontend.

## Table of content
- [Architecture](docs/md/architecture.md)
- [Database](docs/md/database.md)
- [Development environment](#development-environment)
  - [Local running](#local-running)
  - [Local app requests](#local-app-requests)
  - [Tests running](#tests-running)
  - [OpenAPI docs](#openapi-docs)
- [Control by private API](#control-by-private-api)
- [VMaaS](#vmaas)
- [Monitoring](#monitoring)

## Development environment

### Local running
Uses `podman-compose` to deploy the individual project components and supporting containers, which simulate the CMSfR platform and database respectively into local container instance:
~~~bash
podman-compose up --build # Build images if needed and start containers
podman-compose down       # Stop and remove containers
~~~

### Local app requests
When podman compose is running, test app using dev shell scripts:
~~~bash
cd dev/scripts
./systems_list.sh         # show systems
./advisories_list.sh      # show advisories
./platform_sync.sh        # trigger vmaas_sync to sync (using vmaas mock)
./platform_upload.sh      # simulate archive upload to trigger listener and evaluator_upload
~~~

### Tests running
We cover big part of the application functionality with tests and it requires also testing database and some services mocks. It's all encapsulated into the configuration runable using podman-compose command. It also includes static code analysis, database migration tests and dockerfiles checking. It's also used when checking pull requests for the repo.
~~~bash
podman-compose -f docker-compose.test.yml up --build --abort-on-container-exit
~~~

### OpenAPI docs
REST API is documented using OpenAPI v3. For local instance it can be accessed on <http://localhost:8080/openapi/index.html>.

To update/regenerate OpenAPI sources run:
~~~bash
go get -u github.com/swaggo/swag/cmd/swag # download binary to generate, do it first time only
./scripts/generate_docs.sh
~~~

## Control by private API
There is a private API accessible only from inside of `vmaas_sync` container. It allows to run component routines manually. In local environment it can be tested like this:
~~~bash
podman exec -it patchman-engine_vmaas_sync_1 ./sync.sh    # trigger advisories syncing event.
podman exec -it patchman-engine_vmaas_sync_1 ./re-calc.sh # trigger systems recalculation event.
~~~

## VMaaS
This project uses [VMaaS](https://github.com/RedHatInsights/vmaas) for retrieving information about advisories, and resolving which advisories can be applied to which systems. For local development this repo contains VMaaS service mock as a part of platform mock allowing independent running of the service using podman-compose.

## Monitoring
Each application component (except database) exposes metrics for [Prometheus](https://prometheus.io/)
on `/metrics` endpoint (see [docker-compose.yml](docker-compose.yml) for ports). Runtime logs can be send to Amazon
CloudWatch setting proper environment variables (see [awscloudwatch.go](base/utils/awscloudwatch.go)).
