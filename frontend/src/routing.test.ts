import { describe, expect, it } from 'vitest'
import { isGeoPlaygroundPath, isProductMapPath } from './routing'

describe('isGeoPlaygroundPath', () => {
  it.each(['/debug/geo', '/debug/geo/'])('accepts %s', (pathname) => {
    expect(isGeoPlaygroundPath(pathname)).toBe(true)
  })

  it.each(['/', '/map', '/debug/geography'])('rejects %s', (pathname) => {
    expect(isGeoPlaygroundPath(pathname)).toBe(false)
  })
})

describe('isProductMapPath', () => {
  it.each(['/map', '/map/'])('accepts %s', (pathname) => {
    expect(isProductMapPath(pathname)).toBe(true)
  })

  it.each(['/', '/debug/geo', '/maps'])('rejects %s', (pathname) => {
    expect(isProductMapPath(pathname)).toBe(false)
  })
})
