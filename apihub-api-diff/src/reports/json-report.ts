import { DiffResult } from '../schema/diff-result.js'

export const renderJsonReport = (result: DiffResult): string => `${JSON.stringify(result, null, 2)}\n`
