import type { MediaFile } from './media'
import { getMediaByID } from './media'
import { apiFetch } from './client'

export interface Collection {
  id: string
  name: string
  mediaCount: number
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isCollection(value: unknown): value is Collection {
  return isRecord(value) && typeof value.id === 'string' && typeof value.name === 'string' && typeof value.mediaCount === 'number'
}

async function parseCollectionList(response: Response): Promise<Collection[]> {
  if (!response.ok) throw new Error('Impossible de charger les collections')
  const body: unknown = await response.json()
  if (!isRecord(body) || !Array.isArray(body.items) || !body.items.every(isCollection)) throw new Error('La réponse des collections est invalide')
  return body.items
}

export async function getCollections(signal?: AbortSignal): Promise<Collection[]> {
  return parseCollectionList(await apiFetch('/api/collections', signal === undefined ? undefined : { signal }))
}

export async function createCollection(name: string): Promise<Collection> {
  const response = await apiFetch('/api/collections', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) })
  if (!response.ok) throw new Error('Impossible de créer la collection')
  const body: unknown = await response.json()
  if (!isCollection(body)) throw new Error('La réponse de la collection est invalide')
  return body
}

export async function updateCollection(id: string, name: string): Promise<Collection> {
  const response = await apiFetch(`/api/collections/${encodeURIComponent(id)}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) })
  if (!response.ok) throw new Error('Impossible de modifier la collection')
  const body: unknown = await response.json()
  if (!isCollection(body)) throw new Error('La réponse de la collection est invalide')
  return body
}

export async function deleteCollection(id: string): Promise<void> {
  const response = await apiFetch(`/api/collections/${encodeURIComponent(id)}`, { method: 'DELETE' })
  if (!response.ok) throw new Error('Impossible de supprimer la collection')
}

export async function removeMediaFromCollection(collectionId: string, mediaId: string): Promise<void> {
  const response = await apiFetch(`/api/collections/${encodeURIComponent(collectionId)}/media/${encodeURIComponent(mediaId)}`, { method: 'DELETE' })
  if (!response.ok) throw new Error('Impossible de retirer la vidéo de la collection')
}

export async function getMediaCollections(mediaId: string, signal?: AbortSignal): Promise<Collection[]> {
  return parseCollectionList(await apiFetch(`/api/media/${encodeURIComponent(mediaId)}/collections`, signal === undefined ? undefined : { signal }))
}

export async function setMediaCollections(mediaId: string, collectionIds: string[]): Promise<Collection[]> {
  return parseCollectionList(await apiFetch(`/api/media/${encodeURIComponent(mediaId)}/collections`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ collectionIds }) }))
}

export async function getCollectionMedia(collectionId: string, signal?: AbortSignal): Promise<MediaFile[]> {
  const response = await apiFetch(`/api/collections/${encodeURIComponent(collectionId)}/media`, signal === undefined ? undefined : { signal })
  if (!response.ok) throw new Error('Impossible de charger la collection')
  const body: unknown = await response.json()
  if (!isRecord(body) || !Array.isArray(body.mediaIds) || !body.mediaIds.every((id) => typeof id === 'string')) throw new Error('La réponse de la collection est invalide')
  return Promise.all(body.mediaIds.map((id) => getMediaByID(id, signal)))
}
