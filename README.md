[![Tests Status](https://github.com/RedHatInsights/patchman-engine/actions/workflows/unittests.yml/badge.svg)](https://github.com/RedHatInsights/patchman-engine/actions/workflows/unittests.yml)
[![OpenAPI Status](https://github.com/RedHatInsights/patchman-engine/actions/workflows/open_api_spec.yml/badge.svg)](https://github.com/RedHatInsights/patchman-engine/actions/workflows/open_api_spec.yml)
[![Code Coverage](https://codecov.io/gh/RedHatInsights/patchman-engine/branch/master/graph/badge.svg)](https://codecov.io/gh/RedHatInsights/patchman-engine)

# patchman-engine
System Patch Manager is one of the applications for [console.redhat.com](https://console.redhat.com). This application allows users to display and manage available patches for their registered systems. This code repo stores sources for the backend part of the application which provides the REST API to the frontend.

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

### Running locally
Uses `podman-compose` to deploy the individual project components and supporting containers, which simulate the CMSfR platform and database:
~~~bash
podman-compose up --build # Build images if needed and start containers
podman-compose down       # Stop and remove containers
~~~


### Local app requests
When podman compose is running, you can test the app using dev shell scripts:
~~~bash
cd dev/scripts
./systems_list.sh         # show systems
./advisories_list.sh      # show advisories
./platform_sync.sh        # trigger vmaas_sync to sync (using vmaas mock)
./platform_upload.sh      # simulate archive upload to trigger listener and evaluator_upload
~~~

### Running in host OS
Run single component in host OS, rest in podman-compose:
~~~bash
podman-compose stop evaluator_upload # stop single component running using podman-compose
export $(xargs < conf/local.env)
./scripts/entrypoint.sh evaluator # (or listener, or manager) run component in host OS
~~~

### Running tests
We cover a large part of the application functionality with tests; this requires also running a test database and mocked services. This is all encapsulated into the configuration runable using podman-compose command. It also includes static code analysis, database migration tests and dockerfiles checking. It's also used when checking pull requests for the repo.
~~~bash
podman-compose -f docker-compose.test.yml up --build --abort-on-container-exit
~~~

### Run single test
After running all test suit, testing platform components are still running (kafka, platform, db).
So you can run particular test against them, directly from local OS (not from container). This
is especially useful when fixing some test or adding a new one. You need to have golang installed.
~~~bash
export $(xargs < conf/local.env) # setup needed env variables for tests
go test -count=1 -v ./evaluator -run TestEvaluate # run "TestEvaluate" test from "evaluator" component
~~~

### OpenAPI docs
Our REST API is documented using OpenAPI v3. On a local instance it can be accessed on <http://localhost:8080/openapi/index.html>.

To update/regenerate OpenAPI sources run:
~~~bash
go get -u github.com/swaggo/swag/cmd/swag # download binary to generate, do it first time only
./scripts/generate_docs.sh
~~~

## Control by private API
There is a private API accessible only from inside of `vmaas_sync` container. It allows running component routines manually. In local environment it can be tested like this:
~~~bash
podman exec -it patchman-engine_vmaas_sync_1 ./sync.sh    # trigger advisories syncing event.
podman exec -it patchman-engine_vmaas_sync_1 ./re-calc.sh # trigger systems recalculation event.
podman exec -it patchman-engine_vmaas_sync_1 ./caches-check.sh # trigger account caches checking.
~~~

## VMaaS
This project uses [VMaaS](https://github.com/RedHatInsights/vmaas) for retrieving information about advisories, and resolving which advisories can be applied to which systems. For local development this repo contains VMaaS service mock as a part of platform mock allowing independent running of the service using podman-compose.

## Monitoring
Each application component (except for the database) exposes metrics for [Prometheus](https://prometheus.io/)
on `/metrics` endpoint (see [docker-compose.yml](docker-compose.yml) for ports). Runtime logs can be sent to Amazon
CloudWatch if configuration environment variables are set (see [awscloudwatch.go](base/utils/awscloudwatch.go)).

## Kafka control
Your can control and inspect def Kafka instance using:
~~~bash
docker-compose exec kafka bash # enter kafka component and run inside:
/usr/bin/kafka-topics --list --bootstrap-server=kafka:9092 # show created topics

# list all messages send to a topic
/usr/bin/kafka-console-consumer --bootstrap-server=kafka:9092 --topic platform.inventory.events --from-beginning

# send debugging message to a topic
echo '{"id":"00000000-0000-0000-0000-000000000002"}' | /usr/bin/kafka-console-producer --broker-list kafka:9092 --topic patchman.evaluator.upload
~~~

## Run SonarQube code analysis
~~~bash
export SONAR_HOST_URL=https://sonar-server
export SONAR_LOGIN=paste-your-generated-token
export SONAR_CERT_URL=https://secret-url-to/ca.crt # optional
podman-compose -f dev/sonar/docker-compose.yml up --build
~~~

## Update Grafana config map
Copy Grafana board json config to the temporary file, e.g. `grafana.json` and run:
~~~bash
./scripts/grafana-json-to-yaml.sh grafana.json > ./dashboards/grafana-dashboard-insights-patchman-engine-general.configmap.yaml
~~~

## Deps backup
[patchman-engine-deps](https://github.com/RedHatInsights/patchman-engine-deps)
