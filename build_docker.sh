#!/usr/bin/env bash
set -euo pipefail

export CIRCLE_TAG=1.0.0
DOCKER_BUILDKIT=1 docker build --compress --build-arg "VERSION=$CIRCLE_TAG" --tag "gruntwork-io/cloud-nuke:$CIRCLE_TAG" --tag "gruntwork-io/cloud-nuke:latest" .
docker run -it --rm -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY -e AWS_SESSION_TOKEN gruntwork-io/cloud-nuke --version | grep "$CIRCLE_TAG" && echo "TEST OK: Version numbers match up"
