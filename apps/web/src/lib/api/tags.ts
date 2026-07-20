export interface Tag {
  id: string
  name: string
  color: string
}

export interface CreateTagInput {
  name: string
  color: string
}

export interface TagAssignment {
  mediaId: string
  tag: Tag
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isTag(value: unknown): value is Tag {
  return isRecord(value)
    && typeof value.id === 'string'
    && typeof value.name === 'string'
    && typeof value.color === 'string'
}

function isTagAssignment(value: unknown): value is TagAssignment {
  return isRecord(value) && typeof value.mediaId === 'string' && isTag(value.tag)
}

async function parseTagList(response: Response): Promise<Tag[]> {
  if (!response.ok) throw new Error('Impossible de charger les tags')
  const body: unknown = await response.json()
  if (!isRecord(body) || !Array.isArray(body.items) || !body.items.every(isTag)) {
    throw new Error('La réponse des tags est invalide')
  }
  return body.items
}

export async function getTags(signal?: AbortSignal): Promise<Tag[]> {
  const response = await apiFetch('/api/tags', signal === undefined ? undefined : { signal })
  return parseTagList(response)
}

export async function getTagAssignments(signal?: AbortSignal): Promise<TagAssignment[]> {
  const response = await apiFetch('/api/tag-assignments', signal === undefined ? undefined : { signal })
  if (!response.ok) throw new Error('Impossible de charger les attributions de tags')
  const body: unknown = await response.json()
  if (!isRecord(body) || !Array.isArray(body.items) || !body.items.every(isTagAssignment)) {
    throw new Error('La réponse des attributions de tags est invalide')
  }
  return body.items
}

export async function createTag(input: CreateTagInput): Promise<Tag> {
  const response = await apiFetch('/api/tags', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!response.ok) throw new Error('Impossible de créer le tag')
  const body: unknown = await response.json()
  if (!isTag(body)) throw new Error('La réponse du tag est invalide')
  return body
}

export async function getMediaTags(mediaId: string, signal?: AbortSignal): Promise<Tag[]> {
  const response = await apiFetch(`/api/media/${encodeURIComponent(mediaId)}/tags`, signal === undefined ? undefined : { signal })
  return parseTagList(response)
}

export async function setMediaTags(mediaId: string, tagIds: string[]): Promise<Tag[]> {
  const response = await apiFetch(`/api/media/${encodeURIComponent(mediaId)}/tags`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tagIds }),
  })
  return parseTagList(response)
}
import { apiFetch } from './client'
