import { afterEach, describe, expect, it, vi } from 'vitest'
import { getLibraryFolders } from './folders'
afterEach(() => vi.restoreAllMocks())
describe('folders API', () => {
  it('loads relative folder assignments', async () => {
    const items = [{ mediaId: 'media-1', folder: 'Vacances/2026' }]
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ items })))
    await expect(getLibraryFolders('library 1')).resolves.toEqual(items)
    expect(fetchMock).toHaveBeenCalledWith('/api/libraries/library%201/folders', undefined)
  })
})
