[![Build Status](https://travis-ci.org/RedHatInsights/patchman-engine.svg?branch=master)](https://travis-ci.org/RedHatInsights/patchman-engine)
[![Code Coverage](https://codecov.io/gh/RedHatInsights/patchman-engine/branch/master/graph/badge.svg)](https://codecov.io/gh/RedHatInsights/patchman-engine)

# patchman-engine
System Patch Manager application for [cloud.redhat.com](cloud.redhat.com).

## Components
The project is written as a set of communicating containers. The core components are `listener`, `evaluator`, `manager`, `vmaas_sync` and `database` 
- Listener - Connects to kafka service, and listens for messages about newly uploaded hosts.
- Evaluator - Connects to kafka and listents for requests for evaluation, either from `listener` or `vmaas_sync
- Manager - Contains implementation of a REST API, which serves as a primary interface for interacting with the application
- VMaaS sync - Connects to [VMaaS](https://github.com/RedHatInsights/vmaas), and upon receiving notification about updated
 data, syncs new advisories into the database, and requests re-evaluation for systems which could be affected by new advisories
- Database - Self explanatory

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

### Cloud deployment
Relies on the [ocdeployer](https://github.com/bsquizz/ocdeployer) tool. This tool reads templates and supporting configuration files from the `openshift` directory, and
deploys the resulting openshfit templates into specified cluster. 

~~~bash
ocdeployer deploy -t openshift patchman-engine-ci -s build,deploy --secrets-local-dir openshift/secrets -e ./openshift/ci-env.yml
~~~

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
