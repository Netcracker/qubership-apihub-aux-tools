import { createRequire } from 'node:module'
import { join } from 'node:path'
import type {
  BuilderParams,
  BuildConfig,
  BuildResult,
  OperationTypes,
  PackageVersionBuilder as PackageVersionBuilderType,
  VersionDocument,
} from '@netcracker/qubership-apihub-api-processor'

const {
  BUILD_TYPE,
  PackageVersionBuilder,
  VERSION_STATUS,
} = loadApiProcessor()

interface ApiProcessorRuntime {
  BUILD_TYPE: { BUILD: string }
  PackageVersionBuilder: new (config: BuildConfig, params: BuilderParams) => PackageVersionBuilderType
  VERSION_STATUS: { RELEASE: string }
}

export const CLI_PACKAGE_ID = 'apihub-api-diff-cli'
export const PREVIOUS_VERSION = 'previous'
export const CURRENT_VERSION = 'current'

export interface BuildVersionInput {
  fileId: string
  content: Buffer
  version: typeof PREVIOUS_VERSION | typeof CURRENT_VERSION
}

export interface BuiltVersion {
  builder: PackageVersionBuilderType
  result: BuildResult
}

export const buildVersion = async (
  input: BuildVersionInput,
  previous?: BuiltVersion,
): Promise<BuiltVersion> => {
  const config: BuildConfig = {
    packageId: CLI_PACKAGE_ID,
    version: input.version,
    previousVersion: previous ? PREVIOUS_VERSION : undefined,
    previousVersionPackageId: previous ? CLI_PACKAGE_ID : undefined,
    status: VERSION_STATUS.RELEASE,
    buildType: BUILD_TYPE.BUILD,
    files: [
      {
        fileId: input.fileId,
      },
    ],
  }

  const builder = new PackageVersionBuilder(config, {
    configuration: {
      bundleComponents: true,
    },
    resolvers: {
      fileResolver: async (fileId) => {
        if (fileId !== input.fileId) {
          return null
        }
        return new Blob([toArrayBuffer(input.content)])
      },
      versionResolver: async (packageId, version) => {
        if (!previous || packageId !== CLI_PACKAGE_ID || version !== PREVIOUS_VERSION) {
          return null
        }

        return {
          packageId: CLI_PACKAGE_ID,
          version: PREVIOUS_VERSION,
          apiProcessorVersion: API_PROCESSOR_VERSION,
          operationTypes: getOperationTypes(previous.result),
        }
      },
      versionOperationsResolver: async (apiType, version, packageId, operationIds) => {
        if (!previous || packageId !== CLI_PACKAGE_ID || version !== PREVIOUS_VERSION) {
          return null
        }

        const operations = [...previous.result.operations.values()]
          .filter(operation => operation.apiType === apiType)
          .filter(operation => !operationIds || operationIds.includes(operation.operationId))

        return { operations }
      },
      versionDocumentsResolver: async (version, packageId, apiType) => {
        if (!previous || packageId !== CLI_PACKAGE_ID || version !== PREVIOUS_VERSION) {
          return null
        }

        const documents = [...previous.result.documents.values()]
        const apiBuilder = apiType
          ? previous.builder.apiBuilders.find(builder => builder.apiType === apiType)
          : undefined

        return {
          documents: apiBuilder
            ? documents.filter(document => apiBuilder.types.includes(document.type))
            : documents,
          packages: {},
        }
      },
      rawDocumentResolver: async (version, packageId, slug) => {
        if (!previous || packageId !== CLI_PACKAGE_ID || version !== PREVIOUS_VERSION) {
          return null
        }

        const document = [...previous.result.documents.values()].find(document => document.slug === slug)
        if (!document) {
          return null
        }

        return new File([serializePreviousDocument(previous, document)], document.filename)
      },
      versionDeprecatedResolver: async () => ({ operations: [] }),
      versionReferencesResolver: async () => [],
      versionComparisonResolver: async () => null,
    },
  })

  const result = await builder.run({
    withoutChangelog: !previous,
    withoutDeprecatedDepth: true,
  })

  return { builder, result }
}

// Keep this value aligned with the api-processor dependency. The processor validates
// historical versions against its own package version before comparing changelogs.
export const API_PROCESSOR_VERSION = '5.2.1'

const getOperationTypes = (result: BuildResult): OperationTypes[] => {
  const counts = new Map<OperationTypes['apiType'], number>()
  for (const operation of result.operations.values()) {
    counts.set(operation.apiType, (counts.get(operation.apiType) ?? 0) + 1)
  }

  return [...counts.entries()].map(([apiType, operationsCount]) => ({
    apiType,
    operationsCount,
  }))
}

const serializePreviousDocument = (previous: BuiltVersion, document: VersionDocument): BlobPart => {
  const apiBuilder = previous.builder.apiBuilders.find(builder => builder.types.includes(document.type))
  return apiBuilder ? apiBuilder.dumpDocument(document) : ''
}

const toArrayBuffer = (buffer: Buffer): ArrayBuffer => {
  const arrayBuffer = new ArrayBuffer(buffer.byteLength)
  new Uint8Array(arrayBuffer).set(buffer)
  return arrayBuffer
}

function loadApiProcessor(): ApiProcessorRuntime {
  if (typeof require === 'function') {
    // Direct require lets esbuild include api-processor in the SEA bundle.
    return require('@netcracker/qubership-apihub-api-processor') as ApiProcessorRuntime
  }

  const runtimeRequire = createRequire(join(process.cwd(), 'package.json'))
  return runtimeRequire('@netcracker/qubership-apihub-api-processor') as ApiProcessorRuntime
}
