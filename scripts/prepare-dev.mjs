import { mkdir } from 'node:fs/promises'

await mkdir(new URL('../media', import.meta.url), { recursive: true })
