export interface MediaFile {
  id: string
  libraryId: string
  filename: string
  sizeBytes: number
  durationMs: number
  width: number
  height: number
  container: string
  videoCodec: string
  audioCodec: string
  modifiedAt: string
  progressMs: number
  completed: boolean
  title: string
  description: string
  recordedAt: string | null
  favorite: boolean
}

export interface UpdateMediaInput {
  title: string
  description: string
  recordedAt: string | null
  favorite: boolean
}

export interface ScanResult {
  discovered: number
  indexed: number
  unchanged: number
  removed: number
  skipped: number
  issues: ScanIssue[]
}

export interface ScanIssue {
  filename: string
  reason: string
}

export interface ScanStatus {
  state: 'idle' | 'pending' | 'scanning' | 'completed' | 'failed'
  startedAt?: string
  finishedAt?: string
  result?: ScanResult
  lastError?: string
}

export interface PlaybackInfo {
  mode: 'direct' | 'hls'
  url: string
}

export interface HomeMedia {
  continueWatching: MediaFile[]
  recentlyAdded: MediaFile[]
}

export interface MediaSearchResult extends MediaFile {
  libraryName: string
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isMediaFile(value: unknown): value is MediaFile {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.libraryId === 'string' &&
    typeof value.filename === 'string' &&
    typeof value.sizeBytes === 'number' &&
    typeof value.durationMs === 'number' &&
    typeof value.width === 'number' &&
    typeof value.height === 'number' &&
    typeof value.container === 'string' &&
    typeof value.videoCodec === 'string' &&
    typeof value.audioCodec === 'string' &&
    typeof value.modifiedAt === 'string'
    && typeof value.progressMs === 'number'
    && typeof value.completed === 'boolean'
    && typeof value.title === 'string'
    && typeof value.description === 'string'
    && (value.recordedAt === null || typeof value.recordedAt === 'string')
    && typeof value.favorite === 'boolean'
  )
}

function isScanResult(value: unknown): value is ScanResult {
  return (
    isRecord(value) &&
    typeof value.discovered === 'number' &&
    typeof value.indexed === 'number' &&
    typeof value.unchanged === 'number' &&
    typeof value.removed === 'number' &&
    typeof value.skipped === 'number' &&
    Array.isArray(value.issues) &&
    value.issues.every((issue) => isRecord(issue) && typeof issue.filename === 'string' && typeof issue.reason === 'string')
  )
}

function isScanStatus(value: unknown): value is ScanStatus {
  return (
    isRecord(value) &&
    ['idle', 'pending', 'scanning', 'completed', 'failed'].includes(String(value.state)) &&
    (value.startedAt === undefined || typeof value.startedAt === 'string') &&
    (value.finishedAt === undefined || typeof value.finishedAt === 'string') &&
    (value.result === undefined || isScanResult(value.result)) &&
    (value.lastError === undefined || typeof value.lastError === 'string')
  )
}

function isPlaybackInfo(value: unknown): value is PlaybackInfo {
  return isRecord(value) && (value.mode === 'direct' || value.mode === 'hls') && typeof value.url === 'string'
}

function isHomeMedia(value: unknown): value is HomeMedia {
  return (
    isRecord(value) &&
    Array.isArray(value.continueWatching) &&
    value.continueWatching.every(isMediaFile) &&
    Array.isArray(value.recentlyAdded) &&
    value.recentlyAdded.every(isMediaFile)
  )
}

function isMediaSearchResult(value: unknown): value is MediaSearchResult {
  return isRecord(value) && isMediaFile(value) && typeof value.libraryName === 'string'
}

export async function getMedia(libraryId: string, signal?: AbortSignal): Promise<MediaFile[]> {
  const query = new URLSearchParams({ libraryId })
  const response = await apiFetch(`/api/media?${query.toString()}`, signal === undefined ? undefined : { signal })
  if (!response.ok) throw new Error('Impossible de charger les vidéos')

  const body: unknown = await response.json()
  if (!isRecord(body) || !Array.isArray(body.items) || !body.items.every(isMediaFile)) {
    throw new Error('La réponse des vidéos est invalide')
  }
  return body.items
}

export async function getFavorites(signal?: AbortSignal): Promise<MediaFile[]> {
  const response = await apiFetch('/api/favorites', signal === undefined ? undefined : { signal })
  if (!response.ok) throw new Error('Impossible de charger les favoris')
  const body: unknown = await response.json()
  if (!isRecord(body) || !Array.isArray(body.items) || !body.items.every(isMediaFile)) {
    throw new Error('La réponse des favoris est invalide')
  }
  return body.items
}

export async function getMediaByID(mediaId: string, signal?: AbortSignal): Promise<MediaFile> {
  const response = await apiFetch(`/api/media/${encodeURIComponent(mediaId)}`, signal === undefined ? undefined : { signal })
  if (!response.ok) throw new Error('Impossible de charger la vidéo')

  const body: unknown = await response.json()
  if (!isMediaFile(body)) throw new Error('La réponse de la vidéo est invalide')
  return body
}

export async function updateMedia(mediaId: string, input: UpdateMediaInput): Promise<MediaFile> {
  const response = await apiFetch(`/api/media/${encodeURIComponent(mediaId)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!response.ok) throw new Error('Impossible de modifier la vidéo')
  const body: unknown = await response.json()
  if (!isMediaFile(body)) throw new Error('La réponse de modification est invalide')
  return body
}

export async function setMediaFavorite(mediaId: string, favorite: boolean): Promise<MediaFile> {
  const response = await apiFetch(`/api/media/${encodeURIComponent(mediaId)}/favorite`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ favorite }),
  })
  if (!response.ok) throw new Error('Impossible de modifier le favori')
  const body: unknown = await response.json()
  if (!isMediaFile(body)) throw new Error('La réponse vidéo est invalide')
  return body
}

export function thumbnailURL(mediaId: string): string {
  return `/api/media/${encodeURIComponent(mediaId)}/thumbnail`
}

export function streamURL(mediaId: string): string {
  return `/api/media/${encodeURIComponent(mediaId)}/stream`
}

export async function getPlayback(mediaId: string, signal?: AbortSignal): Promise<PlaybackInfo> {
  const response = await apiFetch(`/api/media/${encodeURIComponent(mediaId)}/playback`, signal === undefined ? undefined : { signal })
  if (!response.ok) throw new Error('Impossible de préparer la lecture')

  const body: unknown = await response.json()
  if (!isPlaybackInfo(body)) throw new Error('La réponse de lecture est invalide')
  return body
}

export async function getHomeMedia(signal?: AbortSignal): Promise<HomeMedia> {
  const response = await apiFetch('/api/home', signal === undefined ? undefined : { signal })
  if (!response.ok) throw new Error("Impossible de charger l'accueil")
  const body: unknown = await response.json()
  if (!isHomeMedia(body)) throw new Error("La réponse de l'accueil est invalide")
  return body
}

export async function searchMedia(query: string, signal?: AbortSignal): Promise<MediaSearchResult[]> {
  const parameters = new URLSearchParams({ q: query })
  const response = await apiFetch(`/api/search?${parameters.toString()}`, signal === undefined ? undefined : { signal })
  if (!response.ok) throw new Error('La recherche a échoué')
  const body: unknown = await response.json()
  if (!isRecord(body) || !Array.isArray(body.items) || !body.items.every(isMediaSearchResult)) {
    throw new Error('La réponse de recherche est invalide')
  }
  return body.items
}


export async function scanLibrary(libraryId: string): Promise<ScanResult> {
  const response = await apiFetch(`/api/libraries/${encodeURIComponent(libraryId)}/scan`, { method: 'POST' })
  if (!response.ok) throw new Error("L'analyse de la bibliothèque a échoué")

  const body: unknown = await response.json()
  if (!isScanResult(body)) throw new Error("La réponse d'analyse est invalide")
  return body
}

export async function getScanStatus(libraryId: string, signal?: AbortSignal): Promise<ScanStatus> {
  const response = await apiFetch(
    `/api/libraries/${encodeURIComponent(libraryId)}/scan`,
    signal === undefined ? undefined : { signal },
  )
  if (!response.ok) throw new Error("Impossible de connaître l'état de l'analyse")

  const body: unknown = await response.json()
  if (!isScanStatus(body)) throw new Error("La réponse d'état de l'analyse est invalide")
  return body
}
import { apiFetch } from './client'
