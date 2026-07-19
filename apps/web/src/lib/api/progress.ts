export interface PlaybackProgress {
  mediaId: string
  positionMs: number
  durationMs: number
  completed: boolean
  updatedAt: string
}

export interface SaveProgressInput {
  positionMs: number
  durationMs: number
}

function isProgress(value: unknown): value is PlaybackProgress {
  if (typeof value !== 'object' || value === null) return false
  const item = value as Record<string, unknown>
  return (
    typeof item.mediaId === 'string' &&
    typeof item.positionMs === 'number' &&
    typeof item.durationMs === 'number' &&
    typeof item.completed === 'boolean' &&
    typeof item.updatedAt === 'string'
  )
}

export async function getProgress(mediaId: string, signal?: AbortSignal): Promise<PlaybackProgress> {
  const response = await fetch(`/api/media/${encodeURIComponent(mediaId)}/progress`, signal === undefined ? undefined : { signal })
  if (!response.ok) throw new Error('Impossible de charger la progression')
  const body: unknown = await response.json()
  if (!isProgress(body)) throw new Error('La réponse de progression est invalide')
  return body
}

export async function saveProgress(mediaId: string, input: SaveProgressInput): Promise<PlaybackProgress> {
  const response = await fetch(`/api/media/${encodeURIComponent(mediaId)}/progress`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
    keepalive: true,
  })
  if (!response.ok) throw new Error('Impossible de sauvegarder la progression')
  const body: unknown = await response.json()
  if (!isProgress(body)) throw new Error('La progression sauvegardée est invalide')
  return body
}

