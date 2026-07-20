import { afterEach, describe, expect, it, vi } from 'vitest'

import { changePassword, getAuthStatus, login, logout, setupAdministrator, updateProfile } from './auth'

const user = { id: 'user-1', username: 'admin', role: 'admin' as const, active: true }

afterEach(() => vi.restoreAllMocks())

describe('authentication API', () => {
  it('loads the authentication status', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ configured: true, authenticated: true, user })))
    await expect(getAuthStatus()).resolves.toEqual({ configured: true, authenticated: true, user })
  })

  it('sets up the administrator', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(user), { status: 201 }))
    await expect(setupAdministrator({ username: 'admin', password: 'a secure password' })).resolves.toEqual(user)
    expect(fetchMock).toHaveBeenCalledWith('/api/auth/setup', expect.objectContaining({ method: 'POST' }))
  })

  it('logs in and logs out', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify(user)))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
    await expect(login('admin', 'a secure password')).resolves.toEqual(user)
    await expect(logout()).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/auth/logout', { method: 'POST' })
  })

  it('returns the server login error', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ code: 'invalid_credentials', message: 'Identifiants incorrects' }), { status: 401 }))
    await expect(login('admin', 'wrong')).rejects.toThrow('Identifiants incorrects')
  })

  it('changes the current user password', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 204 }))
    await expect(changePassword('old secure password', 'new secure password')).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenCalledWith('/api/auth/password', expect.objectContaining({ method: 'PUT', body: JSON.stringify({ currentPassword: 'old secure password', newPassword: 'new secure password' }) }))
  })

  it('updates the current profile', async () => {
    const updated = { ...user, username: 'new-admin' }
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(updated)))
    await expect(updateProfile('new-admin')).resolves.toEqual(updated)
    expect(fetchMock).toHaveBeenCalledWith('/api/auth/profile', expect.objectContaining({ method: 'PATCH', body: JSON.stringify({ username: 'new-admin' }) }))
  })
})
