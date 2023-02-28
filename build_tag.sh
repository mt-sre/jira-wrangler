#!/bin/bash

set -exvo pipefail -o nounset

source "${PWD}/cicd/jenkins_env.sh"

export DOCKER_CONFIG="${DOCKER_CONF}"

./mage release
