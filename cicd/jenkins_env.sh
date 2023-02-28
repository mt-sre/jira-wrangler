#!/bin/bash

# utilize local go 1.19 version if available
GO_1_19="/opt/go/1.19.3/bin"

if [ -d  "${GO_1_19}" ]; then
     PATH="${GO_1_19}:${PATH}"
fi

export IMAGE_REGISTRY="quay.io"
export IMAGE_ORG="mtsre"
