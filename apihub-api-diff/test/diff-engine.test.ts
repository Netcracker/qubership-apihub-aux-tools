import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { join } from 'node:path'
import { createDiffResult } from '../src/engine/diff-engine.js'
import { renderHtmlReport } from '../src/reports/html-report.js'
import { renderMarkdownReport } from '../src/reports/markdown-report.js'

const fixture = (name: string): string => join('test', 'fixtures', name)

describe('api diff cli engine', () => {
  it('creates a categorized diff result through api-processor', async () => {
    const result = await createDiffResult({
      previousPath: fixture('openapi-before.yaml'),
      currentPath: fixture('openapi-after.yaml'),
    })

    assert.equal(result.schemaVersion, '1.0')
    assert.equal(result.inputs.previous.fileId, 'openapi-before.yaml')
    assert.ok(result.summary.totalChanges > 0)
    assert.ok(result.changes.length > 0)
    assert.ok(result.changes.some(change => change.apiType === 'rest' && change.diffs.length > 0))
  })

  it('renders markdown and html reports from the normalized result', async () => {
    const result = await createDiffResult({
      previousPath: fixture('openapi-before.yaml'),
      currentPath: fixture('openapi-after.yaml'),
      title: 'Fixture diff',
    })

    assert.match(renderMarkdownReport(result), /# Fixture diff/)
    assert.match(renderHtmlReport(result), /<!doctype html>/)
  })
})
