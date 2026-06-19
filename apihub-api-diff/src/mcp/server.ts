import { createDiffResult } from '../engine/diff-engine.js'
import { renderHtmlReport } from '../reports/html-report.js'
import { renderJsonReport } from '../reports/json-report.js'
import { renderMarkdownReport } from '../reports/markdown-report.js'
import { DiffResult, OutputFormat } from '../schema/diff-result.js'
import { SERVER_INSTRUCTIONS } from './server-instructions.js'

const SERVER_NAME = 'qubership-apihub-api-diff'
const SERVER_VERSION = '0.1.0'
const PROTOCOL_VERSION = '2024-11-05'
const DIFF_TOOL_NAME = 'apihub_api_diff'

type JsonRpcId = string | number | null

interface JsonRpcRequest {
  jsonrpc: '2.0'
  id?: JsonRpcId
  method: string
  params?: any
}

interface JsonRpcResponse {
  jsonrpc: '2.0'
  id: JsonRpcId
  result?: unknown
  error?: {
    code: number
    message: string
    data?: unknown
  }
}

export const startMcpServer = (): void => {
  let buffer = ''

  process.stdin.setEncoding('utf8')
  process.stdin.on('data', chunk => {
    buffer += chunk

    let newlineIndex = buffer.indexOf('\n')
    while (newlineIndex !== -1) {
      const line = buffer.slice(0, newlineIndex).trim()
      buffer = buffer.slice(newlineIndex + 1)

      if (line) {
        processMcpLine(line).then(response => {
          response && writeResponse(response)
        }).catch(error => {
          process.stderr.write(`MCP request failed: ${error instanceof Error ? error.message : String(error)}\n`)
        })
      }

      newlineIndex = buffer.indexOf('\n')
    }
  })
}

export const processMcpLine = async (line: string): Promise<JsonRpcResponse | undefined> => {
  let request: JsonRpcRequest
  try {
    request = JSON.parse(line) as JsonRpcRequest
  } catch (error) {
    return {
      jsonrpc: '2.0',
      id: null,
      error: {
        code: -32700,
        message: 'Parse error',
        data: error instanceof Error ? error.message : String(error),
      },
    }
  }

  if (request.id === undefined) {
    await handleNotification(request)
    return
  }

  try {
    return {
      jsonrpc: '2.0',
      id: request.id,
      result: await handleRequest(request),
    }
  } catch (error) {
    return {
      jsonrpc: '2.0',
      id: request.id,
      error: {
        code: -32603,
        message: error instanceof Error ? error.message : String(error),
      },
    }
  }
}

const handleNotification = async (request: JsonRpcRequest): Promise<void> => {
  if (request.method === 'notifications/initialized' || request.method === 'notifications/cancelled') {
    return
  }

  process.stderr.write(`Ignored MCP notification: ${request.method}\n`)
}

const handleRequest = async (request: JsonRpcRequest): Promise<unknown> => {
  switch (request.method) {
    case 'initialize':
      return initializeResult(request.params)
    case 'ping':
      return {}
    case 'tools/list':
      return toolsListResult()
    case 'tools/call':
      return await callTool(request.params)
    default:
      throw new Error(`Unsupported MCP method: ${request.method}`)
  }
}

const initializeResult = (params: any): unknown => ({
  protocolVersion: typeof params?.protocolVersion === 'string' ? params.protocolVersion : PROTOCOL_VERSION,
  capabilities: {
    tools: {},
  },
  serverInfo: {
    name: SERVER_NAME,
    version: SERVER_VERSION,
  },
  instructions: SERVER_INSTRUCTIONS,
})

const DIFF_TOOL_DESCRIPTION =
  'Compare two API description files (OpenAPI/Swagger YAML or JSON, AsyncAPI, GraphQL SDL, etc.) and return a categorized API changelog from the APIHUB api-processor (severity, action, scope per change). Use this when the user cares about breaking vs non-breaking API changes, migration impact, or a structured review between a baseline spec and a revised spec—not for generic line-by-line file diffs. Prefer format "md" (default) for readable Markdown you can turn into a structured user answer; use "json" with includeValues true for machine-readable output and concrete before/after field values; use "html" only when the user wants a standalone browser report.'

const toolsListResult = (): unknown => ({
  tools: [
    {
      name: DIFF_TOOL_NAME,
      description: DIFF_TOOL_DESCRIPTION,
      inputSchema: {
        type: 'object',
        additionalProperties: false,
        required: ['previousPath', 'currentPath'],
        properties: {
          previousPath: {
            type: 'string',
            description:
              'Absolute or workspace path to the baseline ("old") API document on the machine where this MCP server runs. Ensure the file exists before calling.',
          },
          currentPath: {
            type: 'string',
            description:
              'Absolute or workspace path to the revised ("new") API document on the same machine. Together with previousPath this defines before → after.',
          },
          format: {
            type: 'string',
            enum: ['json', 'md', 'html'],
            default: 'md',
            description:
              'Output shape of the tool response text. Default "md": best for LLM consumption and user-facing summaries (headings, tables). "json": use when you need precise counts, severity buckets, or to chain to other logic; pair with includeValues for raw before/after values. "html": single-page interactive report—only when the user explicitly wants to open it in a browser; avoid for chat-only explanations.',
          },
          includeValues: {
            type: 'boolean',
            default: false,
            description:
              'When true, attaches concrete previous/next values (and richer detail in Markdown when applicable). Enable for deep dive questions ("what exactly changed in this schema?", "show old vs new example"). Increases payload size—summarize for the user unless they asked for full detail.',
          },
          title: {
            type: 'string',
            description:
              'Optional report title shown in the document heading (e.g. release names, ticket id). Helps users map the diff back to a branch, tag, or task.',
          },
        },
      },
    },
  ],
})

const callTool = async (params: any): Promise<unknown> => {
  if (params?.name !== DIFF_TOOL_NAME) {
    throw new Error(`Unknown tool: ${String(params?.name)}`)
  }

  const args = params.arguments ?? {}
  const previousPath = getRequiredString(args, 'previousPath')
  const currentPath = getRequiredString(args, 'currentPath')
  const format = getFormat(args.format)
  const includeValues = Boolean(args.includeValues)
  const title = typeof args.title === 'string' ? args.title : undefined

  const result = await createDiffResult({
    previousPath,
    currentPath,
    includeValues,
    title,
  })

  return {
    content: [
      {
        type: 'text',
        text: renderToolResult(result, format),
      },
    ],
  }
}

const renderToolResult = (result: DiffResult, format: OutputFormat): string => {
  if (format === 'md') {
    return renderMarkdownReport(result)
  }

  if (format === 'html') {
    return renderHtmlReport(result)
  }

  return renderJsonReport(result)
}

const getRequiredString = (args: Record<string, unknown>, name: string): string => {
  const value = args[name]
  if (typeof value !== 'string' || !value) {
    throw new Error(`Tool argument '${name}' is required`)
  }
  return value
}

const getFormat = (value: unknown): OutputFormat => {
  if (value === undefined) {
    return 'md'
  }

  if (value === 'json' || value === 'md' || value === 'html') {
    return value
  }

  throw new Error(`Unsupported format: ${String(value)}`)
}

const writeResponse = (response: JsonRpcResponse): void => {
  process.stdout.write(`${JSON.stringify(response)}\n`)
}
