import { parseArgs } from 'node:util'
import { FailOn, OutputFormat } from '../schema/diff-result.js'

export interface CliOptions {
  previousPath: string
  currentPath: string
  format: OutputFormat
  output?: string
  outputDir?: string
  failOn: FailOn
  includeValues: boolean
  title?: string
  quiet: boolean
}

export const parseCliOptions = (argv: string[]): CliOptions => {
  const { values, positionals } = parseArgs({
    args: argv,
    allowPositionals: true,
    options: {
      format: { type: 'string', short: 'f', default: 'json' },
      output: { type: 'string', short: 'o' },
      'output-dir': { type: 'string' },
      'fail-on': { type: 'string', default: 'never' },
      'include-values': { type: 'boolean', default: false },
      title: { type: 'string' },
      quiet: { type: 'boolean', short: 'q', default: false },
      help: { type: 'boolean', short: 'h', default: false },
    },
  })

  if (values.help) {
    throw new HelpRequested()
  }

  if (positionals.length !== 2) {
    throw new Error('Expected exactly two input files: <previous-file> <current-file>')
  }

  const format = parseEnum('format', values.format, ['json', 'md', 'html'])
  const failOn = parseEnum('fail-on', values['fail-on'], ['breaking', 'risky', 'any', 'never'])

  return {
    previousPath: positionals[0],
    currentPath: positionals[1],
    format,
    output: values.output,
    outputDir: values['output-dir'],
    failOn,
    includeValues: !!values['include-values'],
    title: values.title,
    quiet: !!values.quiet,
  }
}

export class HelpRequested extends Error {}

const parseEnum = <T extends string>(name: string, value: unknown, allowed: readonly T[]): T => {
  if (typeof value === 'string' && allowed.includes(value as T)) {
    return value as T
  }
  throw new Error(`Unsupported ${name}: ${String(value)}. Allowed values: ${allowed.join(', ')}`)
}

export const HELP_TEXT = `Usage:
  apihub-api-diff <previous-file> <current-file> [options]
  apihub-api-diff mcp

Options:
  -f, --format <json|md|html>       Output format. Default: json
  -o, --output <file>               Output file. Default: stdout for json/md, diff.html for html
      --output-dir <directory>      Directory for the default output file
      --fail-on <level>             Exit with code 2 on breaking, risky, any, or never. Default: never
      --include-values              Include raw before/after values in JSON output
      --title <text>                Report title
  -q, --quiet                       Suppress status messages
  -h, --help                        Show this help

MCP:
  Run "apihub-api-diff mcp" to start a local MCP server over stdio.
`
