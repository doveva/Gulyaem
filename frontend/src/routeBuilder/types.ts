import type { FeatureCollection, LineString, MultiLineString } from 'geojson'
import type { CoverageProvenance, CoverageStatus, VersionReference } from '../routeAnalysis'

export interface Waypoint {
  id: string
  lat: number
  lon: number
}

export interface RoutePreview {
  previewFingerprint: string
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

export type WalkStatus = 'DRAFT' | 'ACTIVE' | 'REVIEW' | 'COMPLETED' | 'CANCELLED'

export interface MaterializedRoute {
  id: string
  cityId: string
  geoDataVersionId: string
  profile: 'pedestrian'
  waypoints: Array<{ lat: number; lon: number }>
  geometry: LineString
  distanceMeters: number
  estimatedDurationSeconds: number
  revision: number
}

export interface Walk {
  id: string
  cityId: string
  routeId: string
  status: WalkStatus
  startedAt?: string
  finishedAt?: string
  completedAt?: string
  durationSeconds?: number
  distanceMeters?: number
}

export interface WalkAggregate { walk: Walk; route: MaterializedRoute }

export interface WalkCompletion {
  walk: Walk
  exploration: {
    geoDataVersionId: string
    newSegmentsCount: number
    revisitedSegmentsCount: number
    newNetworkLengthMeters: number
    newSegments: FeatureCollection<LineString>
    districts: Array<{ districtId: string; name: string; percentageBefore: number; percentageAfter: number; newLengthMeters: number }>
  }
}

export interface ExplorationSummary {
  geoDataVersion: { id: string }
  state: { status: 'READY'; updatedAt?: string }
  city: { exploredLengthMeters: number; eligibleLengthMeters: number; percentage: number; exploredSegmentsCount: number }
  districts: Array<{ districtId: string; name: string; exploredLengthMeters: number; eligibleLengthMeters: number; percentage: number }>
}

export interface RoutePreviewErrorPayload {
  code?: string
  message?: string
  error?: { code?: string; message?: string }
}
