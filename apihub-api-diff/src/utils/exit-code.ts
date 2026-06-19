import { DiffResult, FailOn } from '../schema/diff-result.js'

export const resolveExitCode = (result: DiffResult, failOn: FailOn): number => {
  if (failOn === 'never') {
    return 0
  }

  if (failOn === 'any') {
    return result.summary.totalChanges > 0 ? 2 : 0
  }

  if (failOn === 'breaking') {
    return (result.summary.bySeverity.breaking ?? 0) > 0 ? 2 : 0
  }

  const riskyCount = (result.summary.bySeverity.risky ?? 0) + (result.summary.bySeverity['semi-breaking'] ?? 0)
  return riskyCount > 0 || (result.summary.bySeverity.breaking ?? 0) > 0 ? 2 : 0
}
