import type { AuthUser } from './auth'
import { apiFetch } from './client'

export interface CreateUserInput {
  username: string
  password: string
  role: 'admin' | 'user'
}

export interface UpdateUserInput {
  username: string
  role: 'admin' | 'user'
  active: boolean
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isUser(value: unknown): value is AuthUser {
  return isRecord(value) && typeof value.id === 'string' && typeof value.username === 'string' && (value.role === 'admin' || value.role === 'user') && typeof value.active === 'boolean'
}

async function parseError(response: Response): Promise<Error> {
  const body: unknown = await response.json().catch(() => undefined)
  return new Error(isRecord(body) && typeof body.message === 'string' ? body.message : 'Impossible de gérer cet utilisateur')
}

async function parseUser(response: Response): Promise<AuthUser> {
  if (!response.ok) throw await parseError(response)
  const body: unknown = await response.json()
  if (!isUser(body)) throw new Error('La réponse utilisateur est invalide')
  return body
}

export async function getUsers(signal?: AbortSignal): Promise<AuthUser[]> {
  const response = await apiFetch('/api/users', signal === undefined ? undefined : { signal })
  if (!response.ok) throw await parseError(response)
  const body: unknown = await response.json()
  if (!isRecord(body) || !Array.isArray(body.items) || !body.items.every(isUser)) throw new Error('La liste des utilisateurs est invalide')
  return body.items
}

export async function createUser(input: CreateUserInput): Promise<AuthUser> {
  return parseUser(await apiFetch('/api/users', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input) }))
}

export async function updateUser(id: string, input: UpdateUserInput): Promise<AuthUser> {
  return parseUser(await apiFetch(`/api/users/${encodeURIComponent(id)}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input) }))
}

export async function resetUserPassword(id: string, password: string): Promise<void> {
  const response = await apiFetch(`/api/users/${encodeURIComponent(id)}/password`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ password }) })
  if (!response.ok) throw await parseError(response)
}

export async function deleteUser(id: string): Promise<void> {
  const response = await apiFetch(`/api/users/${encodeURIComponent(id)}`, { method: 'DELETE' })
  if (!response.ok) throw await parseError(response)
}
