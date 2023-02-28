#!/bin/bash

set -exvo pipefail -o nounset

source "${PWD}/cicd/jenkins_env.sh"

DOCKER_CONFIG="${DOCKER_CONF}" ./mage release
