export const DIFF_RESULT_SCHEMA_VERSION = '1.0' as const

export type OutputFormat = 'json' | 'md' | 'html'
export type FailOn = 'breaking' | 'risky' | 'any' | 'never'

export interface DiffInput {
  path: string
  fileId: string
  size: number
}

export interface DiffToolInfo {
  name: string
  version: string
}

export interface DiffEngineInfo {
  name: string
  apiProcessorVersion: string
}

export interface DiffSummary {
  totalChanges: number
  bySeverity: Record<string, number>
  byAction: Record<string, number>
  byApiType: Record<string, number>
  changedOperations: number
}

export interface DiffMessage {
  severity: string
  action: string
  scope: string
  description?: string
  previousDeclarationJsonPaths?: unknown[]
  currentDeclarationJsonPaths?: unknown[]
  previousKey?: unknown
  currentKey?: unknown
  previousValue?: unknown
  currentValue?: unknown
}

export interface OperationDiff {
  apiType: string
  operationId?: string
  previousOperationId?: string
  title?: string
  previousTitle?: string
  changeSummary?: unknown
  impactedSummary?: unknown
  diffs: DiffMessage[]
}

export interface ComparisonMetadata {
  packageId: string
  version: string
  previousVersion: string
  previousVersionPackageId: string
  operationTypes: unknown[]
}

export interface DiffNotification {
  severity: unknown
  message: string
  fileId?: string
  operationId?: string
}

export interface DiffResult {
  schemaVersion: typeof DIFF_RESULT_SCHEMA_VERSION
  tool: DiffToolInfo
  engine: DiffEngineInfo
  generatedAt: string
  title: string
  inputs: {
    previous: DiffInput
    current: DiffInput
  }
  summary: DiffSummary
  comparisons: ComparisonMetadata[]
  changes: OperationDiff[]
  notifications: DiffNotification[]
}
