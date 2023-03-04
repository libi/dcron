# stablejob

This **JUST** an example to explain the usage of DCRON stable job.

In this exmaple, we defined a type of job: **execute a bash command** 

## build
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