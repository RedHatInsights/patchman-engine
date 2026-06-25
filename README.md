[![Tests Status](https://github.com/RedHatInsights/patchman-engine/actions/workflows/unittests.yml/badge.svg)](https://github.com/RedHatInsights/patchman-engine/actions/workflows/unittests.yml)
[![OpenAPI Status](https://github.com/RedHatInsights/patchman-engine/actions/workflows/open_api_spec.yml/badge.svg)](https://github.com/RedHatInsights/patchman-engine/actions/workflows/open_api_spec.yml)
[![Code Coverage](https://codecov.io/gh/RedHatInsights/patchman-engine/branch/master/graph/badge.svg)](https://codecov.io/gh/RedHatInsights/patchman-engine)

# patchman-engine
System Patch Manager is one of the applications for [console.redhat.com](https://console.redhat.com). This application allows users to display and manage available patches for their registered systems. This code repo stores sources for the backend part of the application, which provides the REST API for the frontend.

## Table of contents
- [Architecture](docs/md/architecture.md)
- [Database](docs/md/database.md)
- [Development environment](#development-environment)
    - [Running locally](#running-locally)
    - [Local app requests](#local-app-requests)
    - [Testing and debugging](#testing-and-debugging)
    - [OpenAPI docs](#openapi-docs)
- [Control by private API](#control-by-private-api)
- [VMaaS](#vmaas)
- [Monitoring](#monitoring)
- [Profiling](#profiling)

## Development environment
Ensure that you have Go and Podman installed.

### Running locally
Use `podman-compose` (or `docker compose`) to deploy the individual project components and supporting containers that simulate the CMSfR platform and database:
~~~bash
podman-compose up --build # Build images if needed and start containers
podman-compose down       # Stop and remove containers
~~~

#### Run with monitoring
Use `--profile monitoring` to run local `prometheus` and `grafana`, for example:
~~~bash
podman-compose --profile monitoring up
~~~
Grafana is available at <http://localhost:3000> and Prometheus at <http://localhost:9090>.

#### Run a component in the host OS
Run a single component in the host OS while running the rest in podman-compose:
~~~bash
podman-compose stop evaluator_upload # stop a single component using podman-compose
export $(xargs < conf/local.env)
./scripts/entrypoint.sh evaluator # (or listener, or manager) run the component in the host OS
~~~

### Local app requests
When podman-compose is running, use dev shell scripts to test the app:
~~~bash
cd dev/scripts
./systems_list.sh         # show systems
./advisories_list.sh      # show advisories
./platform_sync.sh        # trigger vmaas_sync to sync (using vmaas mock)
./platform_upload.sh      # simulate archive upload to trigger listener and evaluator_upload
~~~

### Testing and debugging
A large part of the application functionality is covered with tests; this requires running a test database and mocked services. All of this is encapsulated in the test configuration, which is run using podman-compose. It also includes static code analysis, database migration tests, and a Dockerfile check. It is used to check pull requests too.

Use this command to run the whole test suite:
~~~bash
podman-compose -f docker-compose.test.yml up --build --abort-on-container-exit
~~~

#### Run one or more tests instead
1. Open `./scripts/go_test.sh` file.
2. Comment out the line that runs all tests.
3. Uncomment and modify the last line to specify one or a set of tests.
4. Run the same command as for the whole suite (from above).

#### Run a single test locally
After running the entire test suite (without `--abort-on-container-exit` flag), testing platform components (`kafka`, `platform`, `db`) are still up. This is especially useful when fixing a test or adding a new one.
~~~bash
podman-compose -f docker-compose.test.yml up --build --no-start # build images
podman-compose -f docker-compose.test.yml run test ./scripts/go_test.sh './evaluator -run TestEvaluate' # run "TestEvaluate" test from "evaluator" component
~~~

#### Run tests in VS Code
A prerequisite is to have the [Go Extension](https://marketplace.visualstudio.com/items?itemName=golang.Go)
installed.

To set things up, copy the example settings from `.vscode/settings.example.json`:
~~~bash
cp .vscode/settings.example.json .vscode/settings.json
~~~

#### Access to the dev/test database
While podman-compose (either dev or test) is running, execute the following to access the database directly:
~~~bash
podman exec -it db psql -d patchman -U admin
~~~

or locally using `psql.sh`, as follows:
~~~bash
export $(cat conf/local.env conf/database_admin.env | xargs ) 2>/dev/null; ./dev/scripts/psql.sh
~~~

#### Kafka control
Control and inspect the Kafka instance using:
~~~bash
podman-compose exec kafka bash # enter kafka component and run inside:
/usr/bin/kafka-topics --list --bootstrap-server=kafka:9092 # show created topics

# list all messages sent to a topic
/usr/bin/kafka-console-consumer --bootstrap-server=kafka:9092 --topic platform.inventory.events --from-beginning

# send debugging message to a topic
echo '{"id":"00000000-0000-0000-0000-000000000002"}' | /usr/bin/kafka-console-producer --broker-list kafka:9092 --topic patchman.evaluator.upload
~~~

### OpenAPI docs
The REST API is documented using OpenAPI v3. On a local instance, it can be accessed at <http://localhost:8080/openapi/index.html>.

For the first time, ensure that you have `swaggo/swag` binary installed:
~~~bash
go get -u github.com/swaggo/swag/cmd/swag
~~~

Run this command to update/regenerate OpenAPI source files:
~~~bash
./scripts/generate_docs.sh
~~~

## Control by private API
There is a private API accessible only from the `vmaas_sync` container, which allows triggering component routines manually. In a local environment, test it like this:
~~~bash
podman exec -it patchman-engine_vmaas_sync_1 ./sync.sh         # trigger advisories syncing event.
podman exec -it patchman-engine_vmaas_sync_1 ./re-calc.sh      # trigger systems recalculation event.
podman exec -it patchman-engine_vmaas_sync_1 ./caches-check.sh # trigger account caches checking.
~~~

## VMaaS
This project uses [VMaaS](https://github.com/RedHatInsights/vmaas) for retrieving information about advisories and resolving which advisories can be applied to which systems. For local development, the platform mock contains a VMaaS service mock.

## Monitoring
Each application component (except for the database) exposes metrics for [Prometheus](https://prometheus.io/)
on `/metrics` endpoint (see [docker-compose.yml](docker-compose.yml) for ports). Runtime logs can be sent to Amazon
CloudWatch if configuration environment variables are set (see [awscloudwatch.go](base/utils/awscloudwatch.go)).

## Run SonarQube code analysis
~~~bash
export SONAR_HOST_URL=https://sonar-server
export SONAR_LOGIN=paste-your-generated-token
export SONAR_CERT_URL=https://secret-url-to/ca.crt # optional
podman-compose -f dev/sonar/docker-compose.yml up --build
~~~

## Update Grafana config map
Copy Grafana board JSON config to a temporary file (e.g. `grafana.json`) and run:
~~~bash
./scripts/grafana-json-to-yaml.sh grafana.json > ./dashboards/app-sre/grafana-dashboard-insights-patchman-engine-general.configmap.yaml
~~~

## Profiling
The app can be profiled using [/net/http/pprof](https://pkg.go.dev/net/http/pprof). Profiler is exposed on app's private port.

### Local development
1. Set `ENABLE_PROFILE=true` in the `conf/common.env`.
2. Run `podman-compose up --build`.
3. Run `go tool pprof http://localhost:{port}/debug/pprof/{heap|profile|block|mutex}` with:
    - `9000` - manager,
    - `9002` - listener,
    - `9003` - evaluator-upload, or
    - `9004` - evaluator-recalc.

### Using Admin API
1. Set `ENABLE_PROFILE_{container_name}=true` in the ClowdApp.
2. Download the profile file using internal API `/api/patch/admin/pprof/{manager|listener|evaluator_upload|evaluator_recalc}/{heap|profile|block|mutex|trace}`.
3. Run `go tool pprof <saved.file>`.
