# patchman-engine
The purpose of this repository is to store source code for System Patch Manager prototype

## Prototypes evaluation

### Go
This is the guide how to test Go prototype:
- Terminal 1
~~~bash
docker-compose up --build db platform # start database and platform-mock
# wait for messages to be send (>>>>>>).
~~~
- Terminal 2
~~~bash
docker-compose up --build go # run go application
# you should see this output:
# {"@timestamp":"2019-11-06T11:48:14Z","duration":0.374642801,"items":30,"levelname":"info","message":"batch finished","write/sec":80.07627510771253}
~~~
- Terminal 3
~~~bash
cd prototypes/go/scripts
./list.sh | grep '"id"' | wc -l # check expected number of returned items (30)
./list.sh # see output
./get_host.sh 1 # get item of id 1, check content
~~~
- Terminal 4
~~~bash
docker-compose up --build ab # run apache benchmark (n - requests, c - parallel)
~~~
