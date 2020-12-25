import { exec } from "https://cdn.depjs.com/exec/mod.ts"

export const root = async () => Deno.chdir(await exec("git rev-parse --show-toplevel"))
export const string = (a: Uint8Array): string => new TextDecoder().decode(a)
export const bytes = (a: string): Uint8Array => new TextEncoder().encode(a)
export const read = async (path: string): Promise<string> =>
  string(await Deno.readFile(path))
export const write = async (path: string, data: string): Promise<void> =>
  Deno.writeFile(path, bytes(data))
