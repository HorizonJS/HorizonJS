#!/bin/bash

docker rm flexjs-extdoc-build
docker build -t flexjs/extdoc-build:latest -f ./Dockerfile-extdoc .

docker run -ti flexjs/extdoc-build:latest /bin/busybox sh