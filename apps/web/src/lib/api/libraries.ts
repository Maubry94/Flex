export interface Library {
  id: string
  name: string
  path: string
  createdAt: string
  lastScanAt: string | null
  lastScanDiscovered: number
  lastScanIndexed: number
  lastScanUnchanged: number
  lastScanSkipped: number
}

interface LibrariesResponse {
  items: Library[]
}

export interface CreateLibraryInput {
  name: string
  path: string
}

interface ApiErrorResponse {
  code: string
  message: string
}

export class ApiError extends Error {
  readonly code: string
  readonly status: number

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isLibrary(value: unknown): value is Library {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.name === 'string' &&
    typeof value.path === 'string' &&
    typeof value.createdAt === 'string'
    && (value.lastScanAt === null || typeof value.lastScanAt === 'string')
    && typeof value.lastScanDiscovered === 'number'
    && typeof value.lastScanIndexed === 'number'
    && typeof value.lastScanUnchanged === 'number'
    && typeof value.lastScanSkipped === 'number'
  )
}

function isLibrariesResponse(value: unknown): value is LibrariesResponse {
  return isRecord(value) && Array.isArray(value.items) && value.items.every(isLibrary)
}

function isApiErrorResponse(value: unknown): value is ApiErrorResponse {
  return isRecord(value) && typeof value.code === 'string' && typeof value.message === 'string'
}

async function parseError(response: Response): Promise<ApiError> {
  const body: unknown = await response.json().catch(() => undefined)
  if (isApiErrorResponse(body)) {
    return new ApiError(response.status, body.code, body.message)
  }
  return new ApiError(response.status, 'unexpected_error', 'Une erreur inattendue est survenue')
}

export async function getLibraries(signal?: AbortSignal): Promise<Library[]> {
  const response = await fetch('/api/libraries', signal === undefined ? undefined : { signal })
  if (!response.ok) throw await parseError(response)

  const body: unknown = await response.json()
  if (!isLibrariesResponse(body)) throw new Error('La réponse des bibliothèques est invalide')
  return body.items
}

export async function createLibrary(input: CreateLibraryInput): Promise<Library> {
  const response = await fetch('/api/libraries', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!response.ok) throw await parseError(response)

  const body: unknown = await response.json()
  if (!isLibrary(body)) throw new Error('La réponse de création est invalide')
  return body
}

export async function updateLibrary(id: string, input: CreateLibraryInput): Promise<Library> {
  const response = await fetch(`/api/libraries/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!response.ok) throw await parseError(response)
  const body: unknown = await response.json()
  if (!isLibrary(body)) throw new Error('La réponse de modification est invalide')
  return body
}

export async function deleteLibrary(id: string): Promise<void> {
  const response = await fetch(`/api/libraries/${encodeURIComponent(id)}`, { method: 'DELETE' })
  if (!response.ok) throw await parseError(response)
}
