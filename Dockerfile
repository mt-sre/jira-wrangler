FROM registry.access.redhat.com/ubi9/go-toolset@sha256:a781bcbb73344bf65c778305c9bdf06c82438fb188d0f2e35e3c3cfb2e834605 AS builder

WORKDIR /workspace

COPY . .

USER root

RUN make

USER default

FROM registry.access.redhat.com/ubi9-micro@sha256:59d5a578037788fafbff25afc985aed9ecfacda4a98608e8425cba7bd3b025da

COPY --from=builder /workspace/bin/jira-wrangler .

USER 65532:65532

ENTRYPOINT ["/jira-wrangler"]
