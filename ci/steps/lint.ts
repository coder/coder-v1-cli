#!/usr/bin/env -S deno run --allow-all
import { cdProjectRoot, execInherit } from "./lib.ts"

await cdProjectRoot()

console.info("--- golangci-lint")
await execInherit("golangci-lint run -c .golangci.yml")
