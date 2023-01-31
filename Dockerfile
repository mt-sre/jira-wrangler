FROM registry.access.redhat.com/ubi9-minimal@sha256:e9ea62ea2017705205ba7bc55d20827e06abe4fe071f0793c6cae46edd5855cf

WORKDIR /app

COPY /jira-wrangler .

USER 65532:65532

ENTRYPOINT ["./jira-wrangler"]
