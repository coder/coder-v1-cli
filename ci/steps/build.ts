#!/usr/bin/env -S deno run --allow-all
import { exec, execInherit, cdProjectRoot, cdCurrent } from "./lib.ts"

const main = async () => {
  await cdProjectRoot()
  Deno.chdir("./ci/steps")

  const goos = Deno.env.get("GOOS")
  const goarch = Deno.env.get("GOARCH") ?? ""

  const tag = await exec("git describe --tags")
  console.info(`--- building coder-cli for ${goos}-${goarch}`)

  const dir = await Deno.makeTempDir()

  await execInherit(
    `go build -ldflags "-X cdr.dev/coder-cli/internal/version.Version=${tag}" -o "${dir}/coder" ../../cmd/coder`
  )

  await Deno.copyFile("../gon.json", `${dir}/gon.json`)
  Deno.chdir(dir)
  switch (goos) {
    case "darwin":
      await packageMacOS(goos, goarch, tag)
      break
    case "windows":
      await packageWindows(goos, goarch, tag)
      break
    case "linux":
      await packageLinux(goos, goarch, tag)
      break
    default:
      throw Error(`unknown GOOS env var: ${goos}`)
  }
  cdCurrent()
  Deno.remove(dir, { recursive: true })
}

const packageWindows = async (goos: string, goarch: string, tag: string) => {
  const artifact = `coder-cli-${goos}-${goarch}-${tag}.zip`
  await Deno.rename("coder", "coder.exe")
  await execInherit(`zip ${artifact} coder.exe`)
}

const packageLinux = async (goos: string, goarch: string, tag: string) => {
  const artifact = `coder-cli-${goos}-${goarch}-${tag}.tar.gz`
  await execInherit(`tar -czf ${artifact} coder`)
}

const packageMacOS = async (goos: string, goarch: string, tag: string) => {
  // cp ../gon.json $tmpdir/gon.json
  const artifact = `coder-cli-${goos}-${goarch}-${tag}.zip`
  await execInherit("gon -log-level debug ./gon.json")
  await Deno.rename("coder.zip", artifact)
}

await main()
