import type { Feature, FeatureCollection, LineString, MultiLineString } from 'geojson'

export type CoverageStatus = 'COMPLETED' | 'PARTIAL' | 'NOT_COVERED' | 'CONNECTOR'
export type CoverageProvenance = 'DIRECT' | 'RADIUS' | 'DIRECT_AND_RADIUS' | ''

export interface SampleRoute {
  id: string
  name: string
  areaId: string
  description: string
  intentionalUnmatched: boolean
  geometry: LineString
}

export interface SampleRouteCollection {
  routes: SampleRoute[]
  geoDataVersion: VersionReference
  expectedSourceChecksum: string
  expectedNormalizationVersion: string
  warnings: string[]
}

export interface VersionReference {
  id: string
  cityId: string
  sourceChecksum: string
  normalizationVersion: string
  status: string
  importedAt: string | null
}

export interface RouteAnalysis {
  routeId: string
  geoDataVersion: VersionReference
  warnings: string[]
  matching: {
    sampleStepMeters: number
    candidateRadiusMeters: number
    maxDirectionDegrees: number
    endpointToleranceMeters: number
  }
  coverageProfile: {
    name: string
    radiusMeters: number
    coverageRatio: number
    minRequiredMeters: number
    maxRequiredMeters: number
  }
  contextRadiusMeters: number
  sourceRoute: LineString
  normalizedRoute: MultiLineString
  matchedFragments: Array<{
    segmentId: string
    classification: 'EXPLORE' | 'ROUTABLE_ONLY'
    reasonCode: string
    geometry: LineString
    routeStartMeters: number
    routeEndMeters: number
    score: {
      distanceScore: number
      directionScore: number
      continuityScore: number
      confidence: number
      reason: string
    }
  }>
  unmatchedFragments: Array<{
    reason: string
    geometry: LineString
    startMeters: number
    endMeters: number
  }>
  coverageSegments: Array<{
    segmentId: string
    classification: 'EXPLORE' | 'ROUTABLE_ONLY'
    geometry: LineString
    lengthMeters: number
    coveredMeters: number
    directMeters: number
    requiredMeters: number
    status: CoverageStatus
    provenance: CoverageProvenance
  }>
  metrics: {
    geometricCoveredLengthMeters: number
    completedNetworkLengthMeters: number
    contextExplorableLengthMeters: number
    completedNetworkRatio: number
    routeMatchedRatio: number
    routeUnmatchedLengthMeters: number
  }
}

export function routeFeature(route: SampleRoute | null): FeatureCollection<LineString> {
  return {
    type: 'FeatureCollection',
    features: route ? [{ type: 'Feature', id: route.id, geometry: route.geometry, properties: { id: route.id } }] : [],
  }
}

export function normalizedFeature(analysis: RouteAnalysis | null): FeatureCollection<MultiLineString> {
  return {
    type: 'FeatureCollection',
    features: analysis ? [{ type: 'Feature', geometry: analysis.normalizedRoute, properties: {} }] : [],
  }
}

export function matchedCollection(analysis: RouteAnalysis | null): FeatureCollection<LineString> {
  return {
    type: 'FeatureCollection',
    features: analysis?.matchedFragments.map((fragment) => ({
      type: 'Feature', id: fragment.segmentId, geometry: fragment.geometry,
      properties: { ...fragment.score, segmentId: fragment.segmentId, classification: fragment.classification },
    })) ?? [],
  }
}

export function unmatchedCollection(analysis: RouteAnalysis | null): FeatureCollection<LineString> {
  return {
    type: 'FeatureCollection',
    features: analysis?.unmatchedFragments.map((fragment, index) => ({
      type: 'Feature', id: `unmatched-${index}`, geometry: fragment.geometry,
      properties: { reason: fragment.reason, startMeters: fragment.startMeters, endMeters: fragment.endMeters },
    })) ?? [],
  }
}

export function coverageCollection(analysis: RouteAnalysis | null): FeatureCollection<LineString> {
  return {
    type: 'FeatureCollection',
    features: analysis?.coverageSegments.map((segment) => ({
      type: 'Feature', id: segment.segmentId, geometry: segment.geometry,
      properties: {
        id: segment.segmentId, status: segment.status, provenance: segment.provenance,
        classification: segment.classification, lengthMeters: segment.lengthMeters,
        coveredMeters: segment.coveredMeters, requiredMeters: segment.requiredMeters,
      },
    })) ?? [],
  }
}

export function routeBounds(route: SampleRoute): [[number, number], [number, number]] {
  const longitudes = route.geometry.coordinates.map((point) => point[0])
  const latitudes = route.geometry.coordinates.map((point) => point[1])
  return [[Math.min(...longitudes), Math.min(...latitudes)], [Math.max(...longitudes), Math.max(...latitudes)]]
}

export type CoverageFeature = Feature<LineString, {
  id: string
  status: CoverageStatus
  provenance: CoverageProvenance
  lengthMeters: number
  coveredMeters: number
  requiredMeters: number
}>
