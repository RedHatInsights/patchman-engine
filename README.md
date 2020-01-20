[![Build Status](https://travis-ci.org/RedHatInsights/patchman-engine.svg?branch=master)](https://travis-ci.org/RedHatInsights/patchman-engine)
[![Code Coverage](https://codecov.io/gh/RedHatInsights/patchman-engine/branch/master/graph/badge.svg)](https://codecov.io/gh/RedHatInsights/patchman-engine)

# patchman-engine
System Patch Manager application for [cloud.redhat.com](cloud.redhat.com).

## Components
The project is written as a set of communicating containers. The core components are `listener`, `evaluator`, `manager` and `database` 
- Listener - Connects to kafka service, and listens for messages.
- Manager - Contains implementation of a REST API, which serves as a primary interface for interacting with the application
- Database - Self explanatory

## Deploying
This project can be deployed either locally or in the cloud using openshift.

### Local deployment
Uses `docker-compose` to deploy the individual project components and supporting containers, which simulate the CMSfR platform and database respectively into local docker instance:
~~~bash
docker-compose up --build # Build images if needed and start containers
docker-compose down       # Stop and remove containers
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

## Test API mockups
Test using dev shell scripts:
~~~bash
# Terminal 1
docker-compose up --build webserver # term 1

# Terminal 2
cd dev/scripts
./advisories_list.sh
./advisory_detail.sh
./systems_applicable.sh 1
./systems_list.sh
./system_detail.sh 1
~~~

Test using Swagger, open <http://localhost:8080/openapi/index.html>.

## Run tests
~~~bash
docker-compose -f docker-compose.test.yml up --build test # Run testing Postgres db and tests against.
~~~
