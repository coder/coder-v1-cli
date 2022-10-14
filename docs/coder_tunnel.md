## coder tunnel

proxies a port on the workspace to localhost

```console
  coder tunnel [workspace_name] [workspace_port] [localhost_port] [flags]
```

### Examples

```console
# run a tcp tunnel from the workspace on port 3000 to localhost:3000
coder tunnel my-dev 3000 3000

# run a udp tunnel from the workspace on port 53 to localhost:53
coder tunnel --udp my-dev 53 53
```

### Options

```console
      --address string                local address to bind to (default "127.0.0.1")
  -h, --help                          help for tunnel
      --max-retry-duration duration   maximum amount of time to sleep between retry attempts (default 1m0s)
      --retry int                     number of attempts to retry if the tunnel fails to establish or disconnect (-1 for infinite retries)
      --udp                           tunnel over UDP instead of TCP
```

### Options inherited from parent commands

```console
      --coder-token string   API authentication token.
      --coder-url string     access url of the Coder deployment. (default "https://demo-2.cdr.dev")
  -v, --verbose              show verbose output (also settable via CODER_AGENT_VERBOSE) (default true)
```
