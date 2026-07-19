import { afterEach, describe, expect, it, vi } from 'vitest'

import { searchMedia } from './media'

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
