import { afterEach, describe, expect, it, vi } from 'vitest'

import { createUser, deleteUser, getUsers, resetUserPassword, updateUser } from './users'

const user = { id: 'user-1', username: 'viewer', role: 'user' as const, active: true }

afterEach(() => vi.restoreAllMocks())

describe('users API', () => {
  it('loads users', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ items: [user] })))
    await expect(getUsers()).resolves.toEqual([user])
  })

  it('creates and updates a user', async () => {
    const updated = { ...user, role: 'admin' as const }
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify(user), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(updated)))
    await expect(createUser({ username: 'viewer', password: 'a secure password', role: 'user' })).resolves.toEqual(user)
    await expect(updateUser(user.id, { username: user.username, role: 'admin', active: true })).resolves.toEqual(updated)
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/users/user-1', expect.objectContaining({ method: 'PATCH' }))
  })

  it('resets a password and deletes a user', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
    await expect(resetUserPassword(user.id, 'a new secure password')).resolves.toBeUndefined()
    await expect(deleteUser(user.id)).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/users/user-1', { method: 'DELETE' })
  })
})
