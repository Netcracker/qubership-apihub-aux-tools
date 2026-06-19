import { createDiffResult } from './engine/diff-engine.js'
import { renderHtmlReport } from './reports/html-report.js'
import { renderJsonReport } from './reports/json-report.js'
import { renderMarkdownReport } from './reports/markdown-report.js'
import { startMcpServer } from './mcp/server.js'
import { HELP_TEXT, HelpRequested, parseCliOptions } from './utils/args.js'
import { resolveExitCode } from './utils/exit-code.js'
import { resolveOutputPath, writeOutput } from './utils/files.js'
import { DiffResult, OutputFormat } from './schema/diff-result.js'

const main = async (): Promise<void> => {
  if (isMcpMode(process.argv.slice(2))) {
    startMcpServer()
    return
  }

  const options = parseCliOptions(process.argv.slice(2))

  if (!options.quiet && options.output) {
    process.stderr.write(`Generating ${options.format} diff report...\n`)
  }

  const result = await createDiffResult({
    previousPath: options.previousPath,
    currentPath: options.currentPath,
    includeValues: options.includeValues,
    title: options.title,
  })

  const rendered = renderReport(result, options.format)
  const outputPath = resolveOutputPath(options.format, options.output, options.outputDir)
  await writeOutput(rendered, outputPath)

  if (!options.quiet && outputPath) {
    process.stderr.write(`Report written to ${outputPath}\n`)
  }

  process.exitCode = resolveExitCode(result, options.failOn)
}

const renderReport = (result: DiffResult, format: OutputFormat): string => {
  if (format === 'md') {
    return renderMarkdownReport(result)
  }

  if (format === 'html') {
    return renderHtmlReport(result)
  }

  return renderJsonReport(result)
}

const isMcpMode = (argv: string[]): boolean => argv[0] === 'mcp' || argv.includes('--mcp')

main().catch(error => {
  if (error instanceof HelpRequested) {
    process.stdout.write(HELP_TEXT)
    return
  }

  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`)
  process.exitCode = 1
})
