# stablejob

This is **JUST** an example to explain the usage of DCRON stable job.
In this exmaple, we defined a type of job: **execute a bash command** .
For run this example, you should install redis, docker and docker-compose first.

## add jobs
1. Start the redis
2. Run the [tools](../tools/tools.go) first, to store the job.

## build and run
```bash
# in dcron srcRoot dir
# build image
docker build -f examples/stablejob/Dockerfile -t stable-job-example .
# run image
docker-compose -f examples/stablejob/stablejob-compose.yml up -d
# scale up
docker-compose -f examples/stablejob/stablejob-compose.yml scale stablejob=5
# get log
docker logs --details [containername]
```