#!/bin/sh
docker build -t redis-proxy . --progress=plain
docker tag redis-proxy:latest hub.pengbei.tech:18080/bigdata/redis-proxy:latest
docker push hub.pengbei.tech:18080/bigdata/redis-proxy:latest