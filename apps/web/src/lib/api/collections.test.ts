import { afterEach, describe, expect, it, vi } from 'vitest'
import { createCollection, deleteCollection, getCollections, getMediaCollections, removeMediaFromCollection, setMediaCollections, updateCollection } from './collections'

const collection = { id: 'collection-1', name: 'Voyages', mediaCount: 2 }
afterEach(() => vi.restoreAllMocks())

describe('collections API', () => {
  it('loads collections', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ items: [collection] })))
    await expect(getCollections()).resolves.toEqual([collection])
  })
  it('creates a collection', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(collection), { status: 201 }))
    await expect(createCollection('Voyages')).resolves.toEqual(collection)
    expect(fetchMock).toHaveBeenCalledWith('/api/collections', expect.objectContaining({ method: 'POST' }))
  })
  it('updates and deletes a collection', async () => {
    const updated = { ...collection, name: 'Escapades' }
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify(updated)))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
    await expect(updateCollection('collection 1', 'Escapades')).resolves.toEqual(updated)
    await expect(deleteCollection('collection 1')).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/collections/collection%201', expect.objectContaining({ method: 'PATCH' }))
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/collections/collection%201', { method: 'DELETE' })
  })
  it('removes media from a collection', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 204 }))
    await expect(removeMediaFromCollection('collection 1', 'media 1')).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenCalledWith('/api/collections/collection%201/media/media%201', { method: 'DELETE' })
  })
  it('loads and updates media collections', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify({ items: [collection] }))).mockResolvedValueOnce(new Response(JSON.stringify({ items: [collection] })))
    await expect(getMediaCollections('media 1')).resolves.toEqual([collection])
    await expect(setMediaCollections('media 1', [collection.id])).resolves.toEqual([collection])
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/media/media%201/collections', expect.objectContaining({ method: 'PUT' }))
  })
})
