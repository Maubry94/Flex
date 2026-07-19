import { describe, expect, it } from 'vitest'

import { mediaTitle } from './media-title'

describe('mediaTitle', () => {
  it.each([
    ['Blue Monday.mp4', 'Blue Monday'],
    ['archive.final.mkv', 'archive.final'],
    ['sans-extension', 'sans-extension'],
    ['.video', '.video'],
  ])('turns %s into %s', (filename, expected) => {
    expect(mediaTitle(filename)).toBe(expected)
  })
})
