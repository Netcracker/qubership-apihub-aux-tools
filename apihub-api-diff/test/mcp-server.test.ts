import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { processMcpLine } from '../src/mcp/server.js'

describe('mcp server protocol', () => {
  it('responds to initialize', async () => {
    const response = await processMcpLine(JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      method: 'initialize',
      params: {},
    }))

    assert.equal(response?.id, 1)
    assert.equal((response?.result as any).serverInfo.name, 'qubership-apihub-api-diff')
    assert.ok((response?.result as any).capabilities.tools)
    const instructions = (response?.result as any).instructions as string
    assert.ok(typeof instructions === 'string' && instructions.includes('When to use'))
  })

  it('lists the API diff tool', async () => {
    const response = await processMcpLine(JSON.stringify({
      jsonrpc: '2.0',
      id: 'tools',
      method: 'tools/list',
      params: {},
    }))

    const tools = (response?.result as any).tools
    assert.equal(tools.length, 1)
    assert.equal(tools[0].name, 'apihub_api_diff')
    assert.ok((tools[0].description as string).includes('OpenAPI'))
    assert.deepEqual(tools[0].inputSchema.required, ['previousPath', 'currentPath'])
    assert.equal(tools[0].inputSchema.properties.format.default, 'md')
  })
})
