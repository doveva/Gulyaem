import type { LineString, MultiLineString } from 'geojson'
import type { CoverageProvenance, CoverageStatus, VersionReference } from '../routeAnalysis'

export interface Waypoint {
  id: string
  lat: number
  lon: number
}

export interface RoutePreview {
  geoDataVersion: VersionReference
  routing: {
    engine: 'valhalla'
    profile: 'pedestrian'
    distanceMeters: number
    durationSeconds: number
    geometry: LineString
    waypoints: Array<{
      input: { lat: number; lon: number }
      resolved?: { lat: number; lon: number }
    }>
  }
  explorationPreview: {
    coverageProfile: {
      name: string
      radiusMeters: number
      coverageRatio: number
      minRequiredMeters: number
      maxRequiredMeters: number
    }
    normalizedRoute: MultiLineString
    matchedFragments: Array<{
      segmentId: string
      classification: 'EXPLORE' | 'ROUTABLE_ONLY'
      geometry: LineString
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
      requiredMeters: number
      status: CoverageStatus
      provenance: CoverageProvenance
    }>
    metrics: {
      routeMatchedRatio: number
      routeUnmatchedLengthMeters: number
      completedNetworkLengthMeters: number
      contextExplorableLengthMeters: number
      completedNetworkRatio: number
      completedSegmentCount: number
      partialSegmentCount: number
      matchedExplorableRouteLengthMeters: number
      matchedRoutableOnlyRouteLengthMeters: number
    }
  }
  warnings: string[]
}

export interface RoutePreviewErrorPayload {
  code?: string
  message?: string
  error?: { code?: string; message?: string }
}
