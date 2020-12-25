#!/usr/bin/env -S deno run --allow-all
import { root, execInherit } from "./lib.ts"

await root()

console.info("--- building integration test image")
await execInherit(
  "docker build -f ./ci/integration/Dockerfile -t coder-cli-integration:latest ."
)

console.info("--- run go tests")
await execInherit("go test ./ci/integration -count=1")
