#!/usr/bin/env -S deno run --allow-all
import { root, execInherit } from "./lib.ts"

await root()

console.info("--- golangci-lint")
await execInherit("golangci-lint run -c .golangci.yml")
