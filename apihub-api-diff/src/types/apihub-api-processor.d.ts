declare module '@netcracker/qubership-apihub-api-processor' {
  export type OperationsApiType = string
  export type OperationId = string
  export type VersionId = string
  export type PackageId = string

  export interface BuildConfig {
    packageId: PackageId
    version: VersionId
    previousVersion?: VersionId
    previousVersionPackageId?: PackageId
    status: string
    buildType: string
    files?: Array<{
      fileId: string
      [key: string]: unknown
    }>
    refs?: Array<{
      refId: string
      version: string
    }>
    metadata?: Record<string, unknown>
    validationRulesSeverity?: Record<string, unknown>
  }

  export interface OperationTypes {
    apiType: OperationsApiType
    operationsCount?: number
    deprecatedCount?: number
    changesSummary?: unknown
  }

  export interface ApiOperation {
    operationId: OperationId
    documentId: string
    title: string
    apiType: OperationsApiType
    apiKind: string
    deprecated: boolean
    metadata: unknown
    data?: unknown
    tags?: string[]
    deprecatedItems?: unknown[]
    deprecatedInfo?: string
    deprecatedInPreviousVersions?: string[]
    models?: Record<string, string>
    apiAudience?: unknown
  }

  export interface VersionDocument {
    fileId: string
    filename: string
    slug: string
    type: string
    format: string
    title: string
    version?: string
    data?: unknown
    apiKind?: string
  }

  export interface VersionsComparison {
    packageId: PackageId
    version: VersionId
    previousVersion: VersionId
    previousVersionPackageId: PackageId
    operationTypes: unknown[]
    data?: unknown[]
    [key: string]: unknown
  }

  export interface BuildResult {
    comparisons: VersionsComparison[]
    notifications: unknown[]
    documents: Map<string, VersionDocument>
    operations: Map<string, ApiOperation>
    exportDocuments: unknown[]
    exportFileName?: string
    config: unknown
    merged?: VersionDocument
  }

  export interface BuilderResolvers {
    fileResolver?: (fileId: string) => Promise<Blob | null>
    versionResolver?: (packageId: PackageId, version: VersionId, includeOperations?: boolean) => Promise<{
      packageId?: PackageId
      version: VersionId
      apiProcessorVersion: string
      operationTypes?: OperationTypes[]
      [key: string]: unknown
    } | null>
    versionOperationsResolver?: (
      apiType: OperationsApiType,
      version: VersionId,
      packageId: PackageId,
      operationIds?: OperationId[],
      includeData?: boolean,
    ) => Promise<{ operations: ApiOperation[] } | null>
    versionDocumentsResolver?: (
      version: VersionId,
      packageId: PackageId,
      apiType?: OperationsApiType,
    ) => Promise<{ documents: VersionDocument[]; packages: Record<string, unknown> } | null>
    rawDocumentResolver?: (version: VersionId, packageId: PackageId, slug: string) => Promise<File | null>
    versionDeprecatedResolver?: (
      apiType: OperationsApiType,
      version?: VersionId,
      packageId?: PackageId,
      operationIds?: OperationId[],
    ) => Promise<{ operations: ApiOperation[] } | null>
    versionReferencesResolver?: (version: VersionId, packageId?: PackageId) => Promise<Array<{ refId: string; version: string }>>
    versionComparisonResolver?: (
      version: VersionId,
      packageId: PackageId,
      previousVersion: VersionId,
      previousVersionPackageId: PackageId,
    ) => Promise<unknown | null>
  }

  export interface BuilderParams {
    resolvers: BuilderResolvers
    configuration?: {
      batchSize?: number
      bundleComponents?: boolean
    }
  }

  export interface BuilderRunOptions {
    withoutChangelog?: boolean
    withoutDeprecatedDepth?: boolean
    cleanCache?: boolean
  }

  export interface ApiBuilder {
    apiType: OperationsApiType
    types: string[]
    dumpDocument: (document: VersionDocument) => Blob
  }

  export class PackageVersionBuilder {
    apiBuilders: ApiBuilder[]
    constructor(config: BuildConfig, params: BuilderParams)
    run(options?: BuilderRunOptions): Promise<BuildResult>
  }

  export const BUILD_TYPE: {
    BUILD: string
  }

  export const VERSION_STATUS: {
    RELEASE: string
  }
}
