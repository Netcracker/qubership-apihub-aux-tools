import { DiffMessage, DiffResult, OperationDiff } from '../schema/diff-result.js'

export const renderMarkdownReport = (result: DiffResult): string => {
  const lines: string[] = [
    `# ${result.title}`,
    '',
    `Generated at: ${result.generatedAt}`,
    '',
    '## Summary',
    '',
    `- Total changes: ${result.summary.totalChanges}`,
    `- Changed operations: ${result.summary.changedOperations}`,
    `- By severity: ${formatCounts(result.summary.bySeverity)}`,
    `- By action: ${formatCounts(result.summary.byAction)}`,
    `- By API type: ${formatCounts(result.summary.byApiType)}`,
    '',
  ]

  if (result.notifications.length) {
    lines.push('## Notifications', '')
    for (const notification of result.notifications) {
      lines.push(`- ${notification.message}`)
    }
    lines.push('')
  }

  lines.push('## Changes', '')

  if (!result.changes.length) {
    lines.push('No changes detected.', '')
    return `${lines.join('\n')}\n`
  }

  for (const change of result.changes) {
    lines.push(...renderOperationChange(change), '')
  }

  return `${lines.join('\n')}\n`
}

const renderOperationChange = (change: OperationDiff): string[] => {
  const operationName = change.operationId ?? change.previousOperationId ?? '<unknown operation>'
  const lines = [
    `### ${operationName}`,
    '',
    `- API type: ${change.apiType}`,
  ]

  if (change.previousOperationId && change.previousOperationId !== change.operationId) {
    lines.push(`- Previous operation ID: ${change.previousOperationId}`)
  }

  lines.push('', '| Severity | Action | Scope | Description |')
  lines.push('| --- | --- | --- | --- |')

  for (const diff of change.diffs) {
    lines.push(`| ${escapeTable(diff.severity)} | ${escapeTable(diff.action)} | ${escapeTable(diff.scope)} | ${escapeTable(diff.description ?? '')} |`)
  }

  const detailBlocks = change.diffs.flatMap((diff, index) => renderDiffDetailBlock(diff, index + 1) ?? [])
  if (detailBlocks.length) {
    lines.push('', '#### Value and pointer detail', '', ...detailBlocks, '')
  }

  return lines
}

const renderDiffDetailBlock = (diff: DiffMessage, index: number): string[] | null => {
  if (!diffHasExtendedDetail(diff)) {
    return null
  }

  const lines: string[] = [`${index}. **${escapeHeading(diff.severity)}** — ${escapeHeading(diff.action)} — ${escapeHeading(diff.scope)}`]

  if (diff.description) {
    lines.push(`   - Description: ${escapeHeading(diff.description)}`)
  }

  if (diff.previousDeclarationJsonPaths?.length) {
    lines.push(`   - Previous JSON paths: \`${escapeInlineJson(diff.previousDeclarationJsonPaths)}\``)
  }

  if (diff.currentDeclarationJsonPaths?.length) {
    lines.push(`   - Current JSON paths: \`${escapeInlineJson(diff.currentDeclarationJsonPaths)}\``)
  }

  if (diff.previousKey !== undefined) {
    lines.push(`   - Previous key: \`${escapeInlineJson(diff.previousKey)}\``)
  }

  if (diff.currentKey !== undefined) {
    lines.push(`   - Current key: \`${escapeInlineJson(diff.currentKey)}\``)
  }

  if (diff.previousValue !== undefined) {
    lines.push('   - Previous value:', ...fenceValue(diff.previousValue))
  }

  if (diff.currentValue !== undefined) {
    lines.push('   - Current value:', ...fenceValue(diff.currentValue))
  }

  lines.push('')
  return lines
}

const diffHasExtendedDetail = (diff: DiffMessage): boolean =>
  diff.previousValue !== undefined
  || diff.currentValue !== undefined
  || Boolean(diff.previousDeclarationJsonPaths?.length)
  || Boolean(diff.currentDeclarationJsonPaths?.length)
  || diff.previousKey !== undefined
  || diff.currentKey !== undefined

const escapeHeading = (value: string): string => value.replace(/\s+/g, ' ').trim()

const escapeInlineJson = (value: unknown): string => JSON.stringify(value).replace(/`/g, '\\`')

const fenceValue = (value: unknown): string[] => {
  const useJsonLabel = value !== null && typeof value === 'object'
  const text = useJsonLabel ? JSON.stringify(value, null, 2) : String(value)
  const clipped = clipForReport(text, 6000)
  const opener = useJsonLabel ? '   ```json' : '   ```'
  return [opener, ...clipped.split('\n').map(line => `   ${line}`), '   ```']
}

const clipForReport = (text: string, maxChars: number): string => {
  if (text.length <= maxChars) {
    return text
  }

  return `${text.slice(0, maxChars)}\n… (${text.length - maxChars} more characters truncated)`
}

const formatCounts = (counts: Record<string, number>): string => {
  const entries = Object.entries(counts)
  return entries.length ? entries.map(([key, value]) => `${key}: ${value}`).join(', ') : 'none'
}

const escapeTable = (value: string): string => value.replace(/\|/g, '\\|').replace(/\n/g, '<br>')
