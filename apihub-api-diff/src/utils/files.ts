import { mkdir, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { OutputFormat } from '../schema/diff-result.js'

export const resolveOutputPath = (
  format: OutputFormat,
  output?: string,
  outputDir?: string,
): string | undefined => {
  if (output) {
    return output
  }

  if (outputDir) {
    return join(outputDir, `diff.${format === 'md' ? 'md' : format}`)
  }

  return format === 'html' ? 'diff.html' : undefined
}

export const writeOutput = async (content: string, outputPath?: string): Promise<void> => {
  if (!outputPath) {
    process.stdout.write(content)
    return
  }

  await mkdir(dirname(outputPath), { recursive: true })
  await writeFile(outputPath, content)
}
