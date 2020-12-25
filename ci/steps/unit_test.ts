#!/usr/bin/env -S deno run --allow-all
import { cdProjectRoot, execInherit } from "./lib.ts"

await cdProjectRoot()

console.info("--- running unit tests")
await execInherit(
  "go test $(go list ./... | grep -v pkg/tcli | grep -v ci/integration | grep -v coder-sdk)"
)
