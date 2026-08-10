import { describe, expect, it } from 'vitest'
import { comparisonCollection, waypointCollection, type RoutingComparison } from './routingComparison'

const comparison = {
  cases: [{
    routeId: 'route', name: 'Route', areaId: 'area', note: '', waypoints: [[30, 60], [31, 61]],
    results: [{
      engineId: 'valhalla', status: 'ok', geometry: { type: 'LineString', coordinates: [[30, 60], [31, 61]] },
      latency: { firstMilliseconds: 1, p50Milliseconds: 1, p95Milliseconds: 2, warmRequests: 30 },
      corridor: { candidateInsideReferenceRatio: 1, referenceInsideCandidateRatio: 1 },
    }],
  }],
} as RoutingComparison

describe('routing comparison GeoJSON', () => {
  it('creates engine lines for the selected case', () => {
    expect(comparisonCollection(comparison, 'route').features[0].properties?.engineId).toBe('valhalla')
    expect(comparisonCollection(comparison, 'missing').features).toHaveLength(0)
  })

  it('numbers shared waypoints', () => {
    expect(waypointCollection(comparison, 'route').features.map((feature) => feature.properties?.index)).toEqual([1, 2])
  })
})
