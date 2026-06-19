import { copyFile, mkdir } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { argv, execPath, platform } from 'node:process'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const outputName = platform === 'win32' ? 'apihub-api-diff.exe' : 'apihub-api-diff'
const binaryPath = join(root, 'dist', outputName)
const blobPath = join(root, 'dist', 'apihub-api-diff.blob')
const postjectCliPath = join(root, 'node_modules', 'postject', 'dist', 'cli.js')

const run = (command, args) => {
  const result = spawnSync(command, args, {
    cwd: root,
    stdio: 'inherit',
    shell: false,
  })

  if (result.status !== 0) {
    throw new Error(`Command failed: ${command} ${args.join(' ')}`)
  }
}

const runNode = (args) => run(execPath, args)

await mkdir(join(root, 'dist'), { recursive: true })
runNode(['--experimental-sea-config', 'sea-config.json'])
await copyFile(execPath, binaryPath)

if (platform === 'darwin') {
  run('codesign', ['--remove-signature', binaryPath])
}

const postjectArgs = [
  binaryPath,
  'NODE_SEA_BLOB',
  blobPath,
  '--sentinel-fuse',
  'NODE_SEA_FUSE_fce680ab2cc467b6e072b8b5df1996b2',
]

if (platform === 'darwin') {
  postjectArgs.push('--macho-segment-name', 'NODE_SEA')
}

runNode([postjectCliPath, ...postjectArgs])

if (platform === 'darwin') {
  run('codesign', ['--sign', '-', binaryPath])
}

console.log(`Created ${binaryPath}`)

if (argv.includes('--print')) {
  console.log(binaryPath)
}
