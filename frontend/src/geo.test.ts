import { describe, expect, it } from 'vitest'
import { endpointCollection, emptyCollection, parseLength, segmentQuery, type SegmentCollection } from './geo'

describe('segmentQuery', () => {
  it('serializes the viewport, classifications, and optional lengths', () => {
    const query = new URLSearchParams(
      segmentQuery([30.3, 59.93, 30.33, 59.945], ['EXPLORE', 'IGNORE'], {
        minLength: 5,
        maxLength: null,
      }),
    )
    expect(query.get('bbox')).toBe('30.300000,59.930000,30.330000,59.945000')
    expect(query.get('classification')).toBe('EXPLORE,IGNORE')
    expect(query.get('minLength')).toBe('5')
    expect(query.has('maxLength')).toBe(false)
  })
})

describe('endpointCollection', () => {
  it('deduplicates endpoints and marks clipped boundary points', () => {
    const collection = emptyCollection() as SegmentCollection
    collection.features = [
      {
        type: 'Feature',
        id: 'one',
        geometry: { type: 'LineString', coordinates: [[30, 60], [31, 61]] },
        properties: {
          id: 'one', geoDataVersionId: 'version', classification: 'EXPLORE', lengthMeters: 1,
          streetName: null, reasonCode: 'test', boundaryClip: true,
        },
      },
    ]
    const points = endpointCollection(collection)
    expect(points.features).toHaveLength(2)
    expect(points.features[0].properties?.kind).toBe('boundary')
  })
})

describe('parseLength', () => {
  it.each([['', null], ['12.5', 12.5], ['-1', 'invalid'], ['x', 'invalid']])('parses %s', (value, result) => {
    expect(parseLength(value)).toBe(result)
  })
})
