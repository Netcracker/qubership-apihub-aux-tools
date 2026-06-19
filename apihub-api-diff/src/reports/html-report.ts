import { DiffResult } from '../schema/diff-result.js'

export const renderHtmlReport = (result: DiffResult): string => {
  const data = escapeScript(JSON.stringify(result))

  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>${escapeHtml(result.title)}</title>
  <style>
    :root { color-scheme: light dark; font-family: Inter, Segoe UI, Arial, sans-serif; }
    body { margin: 0; background: #f6f7fb; color: #172033; }
    header { padding: 28px 36px; background: #18233f; color: white; }
    main { padding: 24px 36px 48px; }
    h1 { margin: 0 0 8px; font-size: 28px; }
    h2 { margin-top: 30px; }
    .muted { color: #667085; }
    header .muted { color: #c9d2e8; }
    .cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; margin: 20px 0; }
    .card { background: white; border-radius: 12px; padding: 16px; box-shadow: 0 1px 3px #1018281a; }
    .card strong { display: block; font-size: 26px; margin-top: 8px; }
    .toolbar { display: flex; gap: 12px; flex-wrap: wrap; margin: 24px 0; }
    input, select { padding: 9px 10px; border: 1px solid #d0d5dd; border-radius: 8px; background: white; color: inherit; }
    .operation { background: white; border-radius: 12px; margin: 14px 0; padding: 18px; box-shadow: 0 1px 3px #1018281a; }
    .operation h3 { margin: 0 0 8px; }
    .diff { display: grid; grid-template-columns: 120px 100px minmax(100px, 1fr) 2fr; gap: 10px; padding: 10px 0; border-top: 1px solid #edf0f5; }
    .badge { display: inline-flex; width: fit-content; border-radius: 999px; padding: 2px 9px; font-size: 12px; font-weight: 600; background: #eef2ff; color: #3538cd; }
    .breaking { background: #fee4e2; color: #b42318; }
    .risky, .semi-breaking { background: #fef0c7; color: #b54708; }
    .non-breaking { background: #dcfae6; color: #067647; }
    .deprecated, .annotation { background: #f2f4f7; color: #344054; }
    @media (prefers-color-scheme: dark) {
      body { background: #0f172a; color: #e5e7eb; }
      header { background: #020617; }
      .card, .operation, input, select { background: #111827; border-color: #334155; }
      .muted { color: #94a3b8; }
      .diff { border-color: #263244; }
    }
  </style>
</head>
<body>
  <header>
    <h1>${escapeHtml(result.title)}</h1>
    <div class="muted">Generated at ${escapeHtml(result.generatedAt)} by ${escapeHtml(result.engine.name)} ${escapeHtml(result.engine.apiProcessorVersion)}</div>
  </header>
  <main>
    <section class="cards">
      <div class="card">Total changes<strong>${result.summary.totalChanges}</strong></div>
      <div class="card">Changed operations<strong>${result.summary.changedOperations}</strong></div>
      <div class="card">Breaking<strong>${result.summary.bySeverity.breaking ?? 0}</strong></div>
      <div class="card">Risky<strong>${(result.summary.bySeverity.risky ?? 0) + (result.summary.bySeverity['semi-breaking'] ?? 0)}</strong></div>
    </section>
    <section class="toolbar">
      <input id="search" type="search" placeholder="Filter operations or descriptions">
      <select id="severity"><option value="">All severities</option></select>
      <select id="apiType"><option value="">All API types</option></select>
    </section>
    <section id="changes"></section>
  </main>
  <script id="diff-data" type="application/json">${data}</script>
  <script>
    const result = JSON.parse(document.getElementById('diff-data').textContent)
    const changesEl = document.getElementById('changes')
    const searchEl = document.getElementById('search')
    const severityEl = document.getElementById('severity')
    const apiTypeEl = document.getElementById('apiType')

    const unique = values => [...new Set(values.filter(Boolean))].sort()
    for (const severity of unique(result.changes.flatMap(change => change.diffs.map(diff => diff.severity)))) {
      severityEl.append(new Option(severity, severity))
    }
    for (const apiType of unique(result.changes.map(change => change.apiType))) {
      apiTypeEl.append(new Option(apiType, apiType))
    }

    const render = () => {
      const query = searchEl.value.toLowerCase()
      const severity = severityEl.value
      const apiType = apiTypeEl.value
      const filtered = result.changes
        .filter(change => !apiType || change.apiType === apiType)
        .map(change => ({ ...change, diffs: change.diffs.filter(diff => !severity || diff.severity === severity) }))
        .filter(change => change.diffs.length)
        .filter(change => !query || JSON.stringify(change).toLowerCase().includes(query))

      changesEl.innerHTML = filtered.length ? filtered.map(renderChange).join('') : '<p class="muted">No changes match the current filters.</p>'
    }

    const esc = value => String(value ?? '').replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]))
    const renderChange = change => '<article class="operation"><h3>' + esc(change.operationId || change.previousOperationId || '<unknown operation>') + '</h3><div class="muted">' + esc(change.apiType) + '</div>' + change.diffs.map(renderDiff).join('') + '</article>'
    const renderDiff = diff => '<div class="diff"><span class="badge ' + esc(diff.severity) + '">' + esc(diff.severity) + '</span><span>' + esc(diff.action) + '</span><span>' + esc(diff.scope) + '</span><span>' + esc(diff.description || '') + '</span></div>'

    searchEl.addEventListener('input', render)
    severityEl.addEventListener('change', render)
    apiTypeEl.addEventListener('change', render)
    render()
  </script>
</body>
</html>
`
}

const escapeHtml = (value: string): string => value.replace(/[&<>"']/g, char => ({
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;',
}[char] ?? char))

const escapeScript = (value: string): string => value.replace(/</g, '\\u003c')
