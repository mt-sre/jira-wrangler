FROM registry.access.redhat.com/ubi9/go-toolset@sha256:a781bcbb73344bf65c778305c9bdf06c82438fb188d0f2e35e3c3cfb2e834605 AS builder

WORKDIR /workspace

COPY . .

USER root

RUN make

USER default

FROM registry.access.redhat.com/ubi9-minimal@sha256:e9ea62ea2017705205ba7bc55d20827e06abe4fe071f0793c6cae46edd5855cf

WORKDIR /app

COPY --from=builder /workspace/bin/jira-wrangler .

USER 65532:65532

ENTRYPOINT ["./jira-wrangler"]
