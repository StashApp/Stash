#!/bin/bash

DOCKER_TAG=$1

# must build the image from dist directory
echo docker build -t stashapp/stash:$DOCKER_TAG -f ./docker/ci/x86_64/Dockerfile ./dist

echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
echo docker push stashapp/stash:$DOCKER_TAG
