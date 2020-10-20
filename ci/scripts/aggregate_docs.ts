// join all /docs files into a single markdown file
// fix all hyperlinks

const newFilename = "coder_cli_all_docs.md"
const newPath = `./docs/${newFilename}`
Deno.chdir(`${new URL(".", import.meta.url).pathname}/../../`)

const dirs = []
for await (const dir of Deno.readDir("./docs")) {
  dirs.push(dir)
}

const filenames = dirs.map(({ name }) => name).filter((f) => f !== newFilename)
const filenameParts = filenames
  .map((f) => f.split("_"))
  .sort((a, b) => a.length - b.length)
  .sort((a, b) => {
    for (let i in a.length > b.length ? a : b) {
      if (a[i] != b[i]) return a > b ? -1 : 1
    }
    return 1
  })

let aggregated = ""
for (let i in filenameParts) {
  const filename = filenameParts[i].join("_")
  const file = await Deno.readFile(`./docs/${filename}`)
  aggregated += `\n${new TextDecoder().decode(file)}`
}
for (let i in filenames) {
  aggregated = aggregated.replaceAll(
    filenames[i],
    `#${filenames[i].replace(".md", "").split("_").join("-")}`
  )
}

try {
  await Deno.remove(newPath)
} catch {}
await Deno.writeFile(newPath, new TextEncoder().encode(aggregated))
