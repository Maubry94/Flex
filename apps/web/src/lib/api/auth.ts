export interface AuthUser {
  id: string
  username: string
  role: 'admin' | 'user'
  active: boolean
}

export interface AuthStatus {
  configured: boolean
  authenticated: boolean
  user?: AuthUser
}

export interface SetupInput {
  username: string
  password: string
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isAuthUser(value: unknown): value is AuthUser {
  return isRecord(value) && typeof value.id === 'string' && typeof value.username === 'string' && (value.role === 'admin' || value.role === 'user') && typeof value.active === 'boolean'
}

async function parseUser(response: Response): Promise<AuthUser> {
  if (!response.ok) {
    const body: unknown = await response.json().catch(() => undefined)
    throw new Error(isRecord(body) && typeof body.message === 'string' ? body.message : 'La requête d’authentification a échoué')
  }
  const body: unknown = await response.json()
  if (!isAuthUser(body)) throw new Error('La réponse d’authentification est invalide')
  return body
}

export async function getAuthStatus(signal?: AbortSignal): Promise<AuthStatus> {
  const response = await apiFetch('/api/auth/status', signal === undefined ? undefined : { signal })
  if (!response.ok) throw new Error('Impossible de vérifier la session')
  const body: unknown = await response.json()
  if (!isRecord(body) || typeof body.configured !== 'boolean' || typeof body.authenticated !== 'boolean' || (body.user !== undefined && !isAuthUser(body.user))) {
    throw new Error('La réponse de session est invalide')
  }
  return { configured: body.configured, authenticated: body.authenticated, ...(body.user === undefined ? {} : { user: body.user }) }
}

export async function setupAdministrator(input: SetupInput): Promise<AuthUser> {
  return parseUser(await apiFetch('/api/auth/setup', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input) }))
}

export async function login(username: string, password: string): Promise<AuthUser> {
  return parseUser(await apiFetch('/api/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, password }) }))
}

export async function logout(): Promise<void> {
  const response = await apiFetch('/api/auth/logout', { method: 'POST' })
  if (!response.ok) throw new Error('Impossible de se déconnecter')
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  const response = await apiFetch('/api/auth/password', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ currentPassword, newPassword }),
  })
  if (!response.ok) {
    const body: unknown = await response.json().catch(() => undefined)
    throw new Error(isRecord(body) && typeof body.message === 'string' ? body.message : 'Impossible de modifier le mot de passe')
  }
}

export async function updateProfile(username: string): Promise<AuthUser> {
  return parseUser(await apiFetch('/api/auth/profile', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username }),
  }))
}
import { apiFetch } from './client'
