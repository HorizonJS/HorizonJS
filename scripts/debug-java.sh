#!/bin/bash

docker rm horizonjs-extdoc-build
docker build -t horizonjs/extdoc-build:latest -f ./Dockerfile-extdoc .

docker run -ti horizonjs/extdoc-build:latest /bin/busybox sh