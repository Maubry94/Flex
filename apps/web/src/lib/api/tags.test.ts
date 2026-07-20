import { afterEach, describe, expect, it, vi } from 'vitest'

import { createTag, getMediaTags, getTagAssignments, getTags, setMediaTags } from './tags'

const tag = { id: 'tag-1', name: 'Voyage', color: '#7c3aed' }

afterEach(() => vi.restoreAllMocks())

describe('tags API', () => {
  it('loads all tags', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ items: [tag] })))
    await expect(getTags()).resolves.toEqual([tag])
    expect(fetchMock).toHaveBeenCalledWith('/api/tags', undefined)
  })

  it('creates a tag', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(tag), { status: 201 }))
    await expect(createTag({ name: tag.name, color: tag.color })).resolves.toEqual(tag)
    expect(fetchMock).toHaveBeenCalledWith('/api/tags', expect.objectContaining({ method: 'POST' }))
  })

  it('loads and updates media tags', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify({ items: [tag] })))
      .mockResolvedValueOnce(new Response(JSON.stringify({ items: [tag] })))
    await expect(getMediaTags('media 1')).resolves.toEqual([tag])
    await expect(setMediaTags('media 1', [tag.id])).resolves.toEqual([tag])
    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/media/media%201/tags', undefined)
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/media/media%201/tags', expect.objectContaining({ method: 'PUT' }))
  })

  it('loads tag assignments', async () => {
    const assignment = { mediaId: 'media-1', tag }
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ items: [assignment] })))
    await expect(getTagAssignments()).resolves.toEqual([assignment])
    expect(fetchMock).toHaveBeenCalledWith('/api/tag-assignments', undefined)
  })
})
