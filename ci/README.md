# ci

## checks

- `steps/build.sh` builds release assets with the appropriate tag injected. Required to pass for merging.
- `steps/fmt.sh` formats all Go source files.
- `steps/gendocs.sh` generates CLI documentation into `/docs` from the command specifications.
- `steps/lint.sh` lints all Go source files based on the rules set fourth in `/.golangci.yml`.


## integration tests

### `tcli`

Package `tcli` provides a framework for writing end-to-end CLI tests.
Each test group can have its own container for executing commands in a consistent
and isolated filesystem.

### running

Assign the following environment variables to run the integration tests
against an existing Enterprise deployment instance.

```bash
export CODER_URL=...
export CODER_EMAIL=...
export CODER_PASSWORD=...
```

Then, simply run the test command from the project root

```sh
./ci/steps/integration.sh
```
