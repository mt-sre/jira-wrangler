# jira-wrangler

A command-line helper for generating progress reports from JIRA.

## Development

### Pre-commit Hooks

Use `./mage hooks:enable` to enable pre-commit hooks locally.
Conversely, `./mage hooks:disable` will disable them.

### Building Locally

Use `./mage build` to build release artifacts locally.
Output binaries will be located in the `dist` folder.

### Testing Locally

Use `./mage test` to run unit tests locally.

Additional static checks can be run with `./mage check`.

### Pushing Images

To push a new _jira-wrangler_ image from your local development environment
you can use the `./mage release:image` command. In order to target a specific
container registry and organization the `IMAGE_REGISTRY` and `IMAGE_ORG`
environment variables must be set before running the command.

Example:

```sh
IMAGE_REGISTRY=quay.io IMAGE_ORG=foobar ./mage release:image
```

### Testing on Kubernetes

To build and apply all artifacts to a kubernetes cluster you can use the
`./mage test:applydev` command. The `IMAGE_REGISTRY` and `IMAGE_ORG`
environment variables must be set to select where the _jira-wrangler_
image will be pushed to and pulled from. Additionally the `JIRA_TOKEN`
environment variable should be set to a valid personal access token
for the JIRA instance you wish to connect to.

Example:

```sh
IMAGE_REGISTRY=quay.io IMAGE_ORG=foobar JIRA_TOKEN=supersecrettoken ./mage test:applydev
```

## License

See [License](LICENSE).
