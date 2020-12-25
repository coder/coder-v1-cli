#!/usr/bin/env -S deno run --allow-all
import { cdProjectRoot, write } from "./lib.ts"

await cdProjectRoot()

interface Params {
  sha: string
  version: string
}

const template = ({ sha, version }: Params) => `class Coder < Formula
  desc "A command-line tool for the Coder remote development platform"
  homepage "https://github.com/cdr/coder-cli"
  url "https://github.com/cdr/coder-cli/releases/download/${version}/coder-cli-darwin-amd64-${version}.zip"
  sha256 "${sha}"

  bottle :unneeded

  def install
    bin.install "coder"
  end

  test do
    system "#{bin}/coder", "--version"
  end
end
`

if (Deno.args.length < 2) {
  throw Error("2 args required")
}

const version = Deno.args[0]
const sha = Deno.args[1]

await write("./coder.rb", template({ sha, version }))
