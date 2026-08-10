import type { FeatureCollection, LineString, Point } from 'geojson'

export const ROUTING_ENGINES = ['valhalla', 'graphhopper', 'osrm'] as const
export type RoutingEngineID = (typeof ROUTING_ENGINES)[number]

export interface RoutingComparison {
  schemaVersion: number
  status: string
  generatedAt: string
  benchmark: { warmRequests: number; corridorMeters: number; sampleStepMeters: number }
  engines: Array<{
    id: RoutingEngineID
    name: string
    version: string
    profile: string
    license: string
    mapMatchingSurface: string
  }>
  setup?: {
    measuredAt: string
    host: Record<string, string>
    engines: Partial<Record<RoutingEngineID, {
      readySeconds: number
      peakMemoryBytes: number
      idleMemoryBytes: number
      graphBytes: number
      graphReused: boolean
    }>>
  }
  cases: RoutingCase[]
  mapMatching: Array<{
    routeId: string
    engineId: RoutingEngineID
    status: string
    error?: string
    routeMatchedRatio?: number
    corridor: {
      candidateInsideReferenceRatio: number
      referenceInsideCandidateRatio: number
    }
  }>
  summary: Array<{
    engineId: RoutingEngineID
    successfulRoutes: number
    totalRoutes: number
    meanCandidateCorridorRatio: number
    meanReferenceCorridorRatio: number
    meanStreetSegmentMatchRatio: number
    medianWarmLatencyMs: number
    mapMatchingStatus: string
  }>
}

export interface RoutingCase {
  routeId: string
  name: string
  areaId: string
  note: string
  waypoints: Array<[number, number]>
  results: RoutingRouteResult[]
}

export interface RoutingRouteResult {
  engineId: RoutingEngineID
  status: string
  error?: string
  distanceMeters?: number
  durationSeconds?: number
  geometryLengthMeters?: number
  responseBytes?: number
  geometry?: LineString
  latency: {
    firstMilliseconds: number
    p50Milliseconds: number
    p95Milliseconds: number
    warmRequests: number
  }
  corridor: {
    candidateInsideReferenceRatio: number
    referenceInsideCandidateRatio: number
  }
  matcher?: {
    routeMatchedRatio: number
    routeUnmatchedLengthMeters: number
    matchedReasonMeters: Record<string, number>
  }
}

export function comparisonCollection(
  comparison: RoutingComparison | null,
  routeId: string,
): FeatureCollection<LineString> {
  const routeCase = comparison?.cases.find((item) => item.routeId === routeId)
  return {
    type: 'FeatureCollection',
    features: routeCase?.results.flatMap((result) => result.geometry ? [{
      type: 'Feature' as const,
      id: `${routeId}-${result.engineId}`,
      geometry: result.geometry,
      properties: { engineId: result.engineId, status: result.status },
    }] : []) ?? [],
  }
}

export function waypointCollection(
  comparison: RoutingComparison | null,
  routeId: string,
): FeatureCollection<Point> {
  const routeCase = comparison?.cases.find((item) => item.routeId === routeId)
  return {
    type: 'FeatureCollection',
    features: routeCase?.waypoints.map((coordinates, index) => ({
      type: 'Feature', id: `${routeId}-waypoint-${index}`,
      geometry: { type: 'Point', coordinates },
      properties: { index: index + 1 },
    })) ?? [],
  }
}
