#!/usr/bin/env -S deno run --allow-all
import { root, execInherit } from "./lib.ts"

await root()

console.info("--- running unit tests")
await execInherit(
  "go test $(go list ./... | grep -v pkg/tcli | grep -v ci/integration | grep -v coder-sdk)"
)
