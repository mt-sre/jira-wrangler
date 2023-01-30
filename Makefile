SHELL=/bin/bash
.SHELLFLAGS=-euo pipefail -c

export CGO_ENABLED=0

build:
	go build -o bin/jira-wrangler ./cmd/jira-wrangler
.PHONY: build

test:
	CGO_ENABLED=1 go test -v -cover -race -count=1 ./...
.PHONY: test

IMAGE_ORG?=quay.io/mt-sre
IMAGE_REPO?=jira-wrangler
IMAGE_TAG?=latest

build-image:
	podman build -f Dockerfile -t "${IMAGE_ORG}/${IMAGE_REPO}:${IMAGE_TAG}" .
.PHONY: build-image

JIRA_URL?=https://issues.redhat.com
JIRA_TOKEN?=token

apply-dev:
	$(eval TMP := $(shell mktemp -d))
	@cp -rT config ${TMP}
	@(cd ${TMP}/overlays/dev \
	  && kustomize edit set image quay.io/mt-sre/jira-wrangler=${IMAGE_ORG}/${IMAGE_REPO}:${IMAGE_TAG} \
	  && kustomize edit add secret jira-wrangler --from-literal jira-url=${JIRA_URL} --from-literal jira-token=${JIRA_TOKEN} \
	  && oc apply -k .\
	)
	@rm -r ${TMP}
.PHONY: apply-dev
