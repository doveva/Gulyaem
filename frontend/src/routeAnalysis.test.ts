import { describe, expect, it } from 'vitest'
import { coverageCollection, routeBounds, type RouteAnalysis, type SampleRoute } from './routeAnalysis'

describe('routeBounds', () => {
  it('finds the extent of a route independently of coordinate order', () => {
    const route = {
      id: 'route', name: 'Route', areaId: 'area', description: '', intentionalUnmatched: false,
      geometry: { type: 'LineString', coordinates: [[30.4, 60.1], [30.2, 59.9], [30.3, 60]] },
    } satisfies SampleRoute
    expect(routeBounds(route)).toEqual([[30.2, 59.9], [30.4, 60.1]])
  })
})

describe('coverageCollection', () => {
  it('keeps status and provenance available to MapLibre layers', () => {
    const analysis = {
      coverageSegments: [{
        segmentId: 'segment', classification: 'EXPLORE',
        geometry: { type: 'LineString', coordinates: [[30, 60], [30.1, 60.1]] },
        lengthMeters: 100, coveredMeters: 70, directMeters: 20, requiredMeters: 60,
        status: 'COMPLETED', provenance: 'DIRECT_AND_RADIUS',
      }],
    } as RouteAnalysis
    const feature = coverageCollection(analysis).features[0]
    expect(feature.properties).toMatchObject({
      id: 'segment', status: 'COMPLETED', provenance: 'DIRECT_AND_RADIUS', coveredMeters: 70,
    })
  })
})
