import { describe, expect, it } from 'vitest'
import { isGeoPlaygroundPath } from './routing'

describe('isGeoPlaygroundPath', () => {
  it.each(['/debug/geo', '/debug/geo/'])('accepts %s', (pathname) => {
    expect(isGeoPlaygroundPath(pathname)).toBe(true)
  })

  it.each(['/', '/map', '/debug/geography'])('rejects %s', (pathname) => {
    expect(isGeoPlaygroundPath(pathname)).toBe(false)
  })
})
