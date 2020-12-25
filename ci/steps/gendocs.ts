#!/usr/bin/env -S deno run --allow-all
import { root, execInherit, requireNoFilesChanged, isCI } from "./lib.ts"

await root()

console.log("--- regenerating documentation")
await Deno.remove("./docs", { recursive: true })
await Deno.mkdir("./docs")

await execInherit("go run ./cmd/coder gen-docs ./docs")
if (isCI()) {
  await requireNoFilesChanged()
}
