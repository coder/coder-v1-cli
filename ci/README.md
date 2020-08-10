# ci

## integration tests

### `tcli`

Package `tcli` provides a framework for writing end-to-end CLI tests.
Each test group can have its own container for executing commands in a consistent
and isolated filesystem.

### prerequisites

Assign the following environment variables to run the integration tests
against an existing Enterprise deployment instance.

```bash
export CODER_URL=...
export CODER_EMAIL=...
export CODER_PASSWORD=...
```

Then, simply run the test command from the project root

```sh
go test -v ./ci/integration
```
