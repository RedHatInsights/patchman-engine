# Containerized Gin application

## Usage
Run testing instance:
~~~bash
docker-compose build
docker-compose up
~~~

## Callings
~~~bash
./health.sh            # check app health
./healthdb.sh          # check app connection to database

./list.sh              # show samples list
./create.sh 100 1.23   # insert new sample id=100, value=1.23
./delete.sh 100        # delete sample of id 100

./metrics.sh           # get Prometheus metrics
~~~

## Tests
Run tests (with sqlite, simple fast, only container):
~~~bash
docker-compose -f docker-compose.test.yml build
docker-compose -f docker-compose.test.yml up
~~~
