import { afterEach, describe, expect, it, vi } from 'vitest'

import { apiFetch, authenticationRequiredEvent } from './client'

afterEach(() => vi.restoreAllMocks())

describe('apiFetch', () => {
  it('notifies the application when a protected request loses its session', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 401 }))
    const listener = vi.fn()
    window.addEventListener(authenticationRequiredEvent, listener)
    await apiFetch('/api/libraries')
    window.removeEventListener(authenticationRequiredEvent, listener)
    expect(listener).toHaveBeenCalledOnce()
  })

  it('does not treat rejected login credentials as an expired session', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 401 }))
    const listener = vi.fn()
    window.addEventListener(authenticationRequiredEvent, listener)
    await apiFetch('/api/auth/login', { method: 'POST' })
    window.removeEventListener(authenticationRequiredEvent, listener)
    expect(listener).not.toHaveBeenCalled()
  })
})
