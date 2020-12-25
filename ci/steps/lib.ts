export const root = async () =>
  Deno.chdir(await exec("git rev-parse --show-toplevel"))

export const string = (a: Uint8Array): string => new TextDecoder().decode(a)

export const bytes = (a: string): Uint8Array => new TextEncoder().encode(a)

export const read = async (path: string): Promise<string> =>
  string(await Deno.readFile(path))

export const write = async (path: string, data: string): Promise<void> =>
  Deno.writeFile(path, bytes(data))

const removeTrailingLineBreak = (str: string) => {
  return str.replace(/\n$/, "")
}

export const execInherit = async (
  cmd: string | string[] | ExecOptions
): Promise<void> => {
  let opts: Deno.RunOptions
  if (typeof cmd === "string") {
    opts = {
      cmd: ["sh", "-c", cmd],
    }
  } else if (Array.isArray(cmd)) {
    opts = {
      cmd,
    }
  } else {
    opts = cmd
  }

  opts.stdout = "inherit"
  opts.stderr = "inherit"

  const process = Deno.run(opts)
  const { success } = await process.status()
  if (!success) {
    process.close()
    throw new Error("exec: failed to execute command")
  }
}

export type ExecOptions = Omit<Deno.RunOptions, "stdout" | "stderr">

export const exec = async (
  cmd: string | string[] | ExecOptions
): Promise<string> => {
  let opts: Deno.RunOptions

  if (typeof cmd === "string") {
    opts = {
      cmd: ["sh", "-c", cmd],
    }
  } else if (Array.isArray(cmd)) {
    opts = {
      cmd,
    }
  } else {
    opts = cmd
  }

  opts.stdout = "piped"
  opts.stderr = "piped"

  const process = Deno.run(opts)
  const decoder = new TextDecoder()
  const { success } = await process.status()

  if (!success) {
    const msg = removeTrailingLineBreak(
      decoder.decode(await process.stderrOutput())
    )

    process.close()

    throw new Error(msg || "exec: failed to execute command")
  }

  return removeTrailingLineBreak(decoder.decode(await process.output()))
}

export const requireNoFilesChanged = async (): Promise<void> => {
  const changed = await exec(
    "git ls-files --other --modified --exclude-standard"
  )
  if (changed !== "") {
    throw Error(`Files needs generation or formatting:
${changed}`)
  }
}

export const isCI = (): boolean => !!Deno.env.get("CI")
