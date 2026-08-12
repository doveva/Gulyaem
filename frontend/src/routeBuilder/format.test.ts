import { describe, expect, it } from 'vitest'
import { formatDistance, formatDuration } from './format'

describe('route summary formatting', () => {
  it('formats distance for compact Russian UI', () => {
    expect(formatDistance(420)).toBe('420 м')
    expect(formatDistance(4200)).toBe('4,2 км')
  })

  it('formats walking duration', () => {
    expect(formatDuration(3600)).toBe('1 ч')
    expect(formatDuration(4500)).toBe('1 ч 15 мин')
  })
})
