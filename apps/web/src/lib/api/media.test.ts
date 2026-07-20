import { afterEach, describe, expect, it, vi } from 'vitest'

import { getFavorites, searchMedia, updateMedia } from './media'

const searchResult = {
  id: 'media-1',
  libraryId: 'library-1',
  libraryName: 'AMVs',
  filename: 'Blue Monday.mp4',
  sizeBytes: 1_024,
  durationMs: 30_000,
  width: 1920,
  height: 1080,
  container: 'mp4',
  videoCodec: 'h264',
  audioCodec: 'aac',
  modifiedAt: '2026-07-20T00:00:00Z',
  progressMs: 0,
  completed: false,
  title: 'Blue Monday',
  description: '',
  recordedAt: null,
  favorite: false,
}

describe('searchMedia', () => {
  afterEach(() => vi.restoreAllMocks())

  it('encodes the query and validates search results', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ items: [searchResult] }), { status: 200 }),
    )

    await expect(searchMedia('Blue Monday')).resolves.toEqual([searchResult])
    expect(fetchMock).toHaveBeenCalledWith('/api/search?q=Blue+Monday', undefined)
  })

  it('rejects a result without its library name', async () => {
    const invalidResult: Record<string, unknown> = { ...searchResult }
    delete invalidResult.libraryName
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ items: [invalidResult] }), { status: 200 }),
    )

    await expect(searchMedia('Blue')).rejects.toThrow('invalide')
  })
})

describe('updateMedia', () => {
  afterEach(() => vi.restoreAllMocks())

  it('updates editorial metadata without changing the file', async () => {
    const updated = { ...searchResult, description: 'Une vidéo personnelle', favorite: true }
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify(updated), { status: 200 }),
    )

    await expect(updateMedia(searchResult.id, {
      title: updated.title,
      description: updated.description,
      recordedAt: null,
      favorite: true,
    })).resolves.toEqual(updated)
    expect(fetchMock).toHaveBeenCalledWith('/api/media/media-1', expect.objectContaining({ method: 'PATCH' }))
  })
})

describe('getFavorites', () => {
  afterEach(() => vi.restoreAllMocks())

  it('returns the favorite media list', async () => {
    const favorite = { ...searchResult, favorite: true }
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ items: [favorite] }), { status: 200 }),
    )

    await expect(getFavorites()).resolves.toEqual([favorite])
    expect(fetchMock).toHaveBeenCalledWith('/api/favorites', undefined)
  })

  it('rejects an invalid favorite response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ items: [{ id: 'incomplete' }] }), { status: 200 }),
    )

    await expect(getFavorites()).rejects.toThrow('invalide')
  })
})
