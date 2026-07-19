import { afterEach, describe, expect, it, vi } from 'vitest'

import { getHealth } from './health'

describe('getHealth', () => {
  afterEach(() => vi.restoreAllMocks())

  it('returns a validated health response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ status: 'ok', service: 'flex' }), { status: 200 }),
    )

    await expect(getHealth()).resolves.toEqual({ status: 'ok', service: 'flex' })
  })

  it('rejects an invalid response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ status: 'maybe' }), { status: 200 }),
    )

    await expect(getHealth()).rejects.toThrow('invalide')
  })
})

