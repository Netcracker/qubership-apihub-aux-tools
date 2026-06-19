import { readFile, stat } from 'node:fs/promises'
import { basename, resolve } from 'node:path'
import { API_PROCESSOR_VERSION, buildVersion, CURRENT_VERSION, PREVIOUS_VERSION } from './processor-builder.js'
import {
  DIFF_RESULT_SCHEMA_VERSION,
  DiffInput,
  DiffMessage,
  DiffNotification,
  DiffResult,
  DiffSummary,
  OperationDiff,
} from '../schema/diff-result.js'

export interface DiffEngineOptions {
  previousPath: string
  currentPath: string
  includeValues?: boolean
  title?: string
}

const TOOL_NAME = 'apihub-api-diff'
const TOOL_VERSION = '0.1.0'

export const createDiffResult = async (options: DiffEngineOptions): Promise<DiffResult> => {
  const previousInput = await readInput(options.previousPath)
  const currentInput = await readInput(options.currentPath)

  const previous = await buildVersion({
    fileId: previousInput.fileId,
    content: previousInput.content,
    version: PREVIOUS_VERSION,
  })

  const current = await buildVersion({
    fileId: currentInput.fileId,
    content: currentInput.content,
    version: CURRENT_VERSION,
  }, previous)

  const changes = current.result.comparisons.flatMap(comparison => normalizeComparisonChanges(comparison, !!options.includeValues))
  const notifications = [
    ...normalizeNotifications(previous.result.notifications),
    ...normalizeNotifications(current.result.notifications),
  ]

  return {
    schemaVersion: DIFF_RESULT_SCHEMA_VERSION,
    tool: {
      name: TOOL_NAME,
      version: TOOL_VERSION,
    },
    engine: {
      name: '@netcracker/qubership-apihub-api-processor',
      apiProcessorVersion: API_PROCESSOR_VERSION,
    },
    generatedAt: new Date().toISOString(),
    title: options.title ?? `${basename(options.previousPath)} -> ${basename(options.currentPath)}`,
    inputs: {
      previous: previousInput.meta,
      current: currentInput.meta,
    },
    summary: createSummary(changes),
    comparisons: current.result.comparisons.map(comparison => ({
      packageId: comparison.packageId,
      version: comparison.version,
      previousVersion: comparison.previousVersion,
      previousVersionPackageId: comparison.previousVersionPackageId,
      operationTypes: comparison.operationTypes,
    })),
    changes,
    notifications,
  }
}

interface ReadInputResult {
  content: Buffer
  fileId: string
  meta: DiffInput
}

const readInput = async (path: string): Promise<ReadInputResult> => {
  const absolutePath = resolve(path)
  const [content, stats] = await Promise.all([
    readFile(absolutePath),
    stat(absolutePath),
  ])
  const fileId = basename(path)

  return {
    content,
    fileId,
    meta: {
      path: absolutePath,
      fileId,
      size: stats.size,
    },
  }
}

const normalizeComparisonChanges = (comparison: any, includeValues: boolean): OperationDiff[] => {
  const changes = Array.isArray(comparison.data) ? comparison.data : []

  return changes.map((change: any) => ({
    apiType: String(change.apiType ?? ''),
    operationId: change.operationId,
    previousOperationId: change.previousOperationId,
    title: change.metadata?.path ?? change.metadata?.method,
    previousTitle: change.previousMetadata?.path ?? change.previousMetadata?.method,
    changeSummary: change.changeSummary,
    impactedSummary: change.impactedSummary,
    diffs: normalizeDiffs(change.diffs, includeValues),
  }))
}

const normalizeDiffs = (diffs: unknown, includeValues: boolean): DiffMessage[] => {
  if (!Array.isArray(diffs)) {
    return []
  }

  return diffs.map((diff: any) => {
    const normalized: DiffMessage = {
      severity: String(diff.type ?? diff.severity ?? 'unclassified'),
      action: String(diff.action ?? ''),
      scope: String(diff.scope ?? ''),
      description: diff.description,
      previousDeclarationJsonPaths: diff.beforeDeclarationPaths,
      currentDeclarationJsonPaths: diff.afterDeclarationPaths,
      previousKey: diff.beforeKey,
      currentKey: diff.afterKey,
    }

    if (includeValues) {
      normalized.previousValue = toJsonSafeValue(diff.beforeValue)
      normalized.currentValue = toJsonSafeValue(diff.afterValue)
    }

    return normalized
  })
}

const normalizeNotifications = (notifications: unknown[]): DiffNotification[] => notifications.map((notification: any) => ({
  severity: notification.severity,
  message: String(notification.message ?? ''),
  fileId: notification.fileId,
  operationId: notification.operationId,
}))

const createSummary = (changes: OperationDiff[]): DiffSummary => {
  const diffs = changes.flatMap(change => change.diffs)

  return {
    totalChanges: diffs.length,
    bySeverity: countBy(diffs, diff => diff.severity),
    byAction: countBy(diffs, diff => diff.action),
    byApiType: countBy(changes, change => change.apiType),
    changedOperations: changes.length,
  }
}

const countBy = <T>(items: T[], keySelector: (item: T) => string): Record<string, number> => {
  const counts: Record<string, number> = {}
  for (const item of items) {
    const key = keySelector(item) || 'unknown'
    counts[key] = (counts[key] ?? 0) + 1
  }
  return counts
}

const toJsonSafeValue = (value: unknown): unknown => {
  if (value === undefined) {
    return undefined
  }

  try {
    return JSON.parse(JSON.stringify(value))
  } catch {
    return String(value)
  }
}
