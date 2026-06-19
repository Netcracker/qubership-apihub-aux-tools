import { build } from 'esbuild'
import { chmod } from 'node:fs/promises'

await build({
  entryPoints: ['src/cli.ts'],
  outfile: 'dist/cli.cjs',
  bundle: true,
  platform: 'node',
  target: 'node20',
  format: 'cjs',
  banner: {
    js: '#!/usr/bin/env node',
  },
  sourcemap: true,
  legalComments: 'external',
})

await chmod('dist/cli.cjs', 0o755)
