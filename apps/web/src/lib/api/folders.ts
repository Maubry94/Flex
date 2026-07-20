export interface FolderAssignment {
  mediaId: string
  folder: string
}

function isAssignment(value: unknown): value is FolderAssignment {
  return typeof value === 'object' && value !== null
    && typeof (value as Record<string, unknown>).mediaId === 'string'
    && typeof (value as Record<string, unknown>).folder === 'string'
}

export async function getLibraryFolders(libraryId: string, signal?: AbortSignal): Promise<FolderAssignment[]> {
  const response = await apiFetch(`/api/libraries/${encodeURIComponent(libraryId)}/folders`, signal === undefined ? undefined : { signal })
  if (!response.ok) throw new Error('Impossible de charger les dossiers')
  const body: unknown = await response.json()
  if (typeof body !== 'object' || body === null || !Array.isArray((body as Record<string, unknown>).items) || !(body as { items: unknown[] }).items.every(isAssignment)) {
    throw new Error('La réponse des dossiers est invalide')
  }
  return (body as { items: FolderAssignment[] }).items
}
import { apiFetch } from './client'
