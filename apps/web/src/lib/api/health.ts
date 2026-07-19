export interface HealthResponse {
  status: 'ok'
  service: 'flex'
}

function isHealthResponse(value: unknown): value is HealthResponse {
  if (typeof value !== 'object' || value === null) return false

  const candidate = value as Record<string, unknown>
  return candidate.status === 'ok' && candidate.service === 'flex'
}

export async function getHealth(signal?: AbortSignal): Promise<HealthResponse> {
  const response = await fetch('/api/health', signal === undefined ? undefined : { signal })
  if (!response.ok) {
    throw new Error(`Le serveur a répondu avec le statut ${String(response.status)}`)
  }

  const body: unknown = await response.json()
  if (!isHealthResponse(body)) {
    throw new Error('La réponse du serveur est invalide')
  }
  return body
}
