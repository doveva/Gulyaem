export const CITY_ID =
  import.meta.env.VITE_CITY_ID ?? '01900000-0000-7000-8000-000000000001'

export const CLASSIFICATIONS = ['EXPLORE', 'ROUTABLE_ONLY', 'IGNORE'] as const
export type Classification = (typeof CLASSIFICATIONS)[number]

export interface GeoVersion {
  id: string
  cityId: string
  source: string
  sourceTimestamp: string | null
  sourceChecksum: string
  normalizationVersion: string
  status: string
  importedAt: string | null
}

export interface SegmentProperties {
  id: string
  geoDataVersionId: string
  classification: Classification
  lengthMeters: number
  streetName: string | null
  reasonCode: string
  boundaryClip: boolean
}

export type SegmentFeature = Feature<LineString, SegmentProperties>

export interface SegmentStatistics {
  segmentsTotal: number
  exploreCount: number
  routableOnlyCount: number
  ignoreCount: number
  totalLengthMeters: number
  explorableLengthMeters: number
  minLengthMeters: number
  medianLengthMeters: number
  p95LengthMeters: number
  maxLengthMeters: number
  shortSegmentCount: number
  longSegmentCount: number
}

export interface SegmentCollection extends FeatureCollection<LineString, SegmentProperties> {
  meta: {
    geoDataVersionId: string
    returnedCount: number
    bbox: [number, number, number, number]
    statistics: SegmentStatistics
  }
}

export interface SegmentDetail {
  id: string
  cityId: string
  geoDataVersionId: string
  versionStatus: string
  normalizationVersion: string
  isCurrent: boolean
  geometry: LineString
  lengthMeters: number
  classification: Classification
  reasonCode: string
  normalization: {
    boundaryClipped: boolean
    warnings: string[]
  }
  street: { id: string; name: string | null } | null
  districts: Array<{
    id: string
    districtDataVersionId: string
    name: string
    kind: string
  }>
  debugSource?: {
    tags?: Record<string, string>
    wayIds?: number[]
    startNodeId?: number
    endNodeId?: number
  }
}

export interface DistrictProperties {
  id: string
  districtDataVersionId: string
  externalId: string
  name: string
  kind: string
  labelPoint: Point
  source: string
  sourceTimestamp: string | null
  normalizationVersion: string
}

export type DistrictFeature = Feature<Polygon | MultiPolygon, DistrictProperties>

export interface DistrictCollection extends FeatureCollection<Polygon | MultiPolygon, DistrictProperties> {
  meta: {
    districtDataVersionId: string
    returnedCount: number
    bbox: [number, number, number, number]
  }
}

export interface AppliedFilters {
  minLength: number | null
  maxLength: number | null
}

export interface APIErrorPayload {
  error?: { code?: string; message?: string }
}

export const EMPTY_STATISTICS: SegmentStatistics = {
  segmentsTotal: 0,
  exploreCount: 0,
  routableOnlyCount: 0,
  ignoreCount: 0,
  totalLengthMeters: 0,
  explorableLengthMeters: 0,
  minLengthMeters: 0,
  medianLengthMeters: 0,
  p95LengthMeters: 0,
  maxLengthMeters: 0,
  shortSegmentCount: 0,
  longSegmentCount: 0,
}

export function emptyCollection(): SegmentCollection {
  return {
    type: 'FeatureCollection',
    features: [],
    meta: {
      geoDataVersionId: '',
      returnedCount: 0,
      bbox: [0, 0, 0, 0],
      statistics: EMPTY_STATISTICS,
    },
  }
}

export function emptyDistrictCollection(): DistrictCollection {
  return {
    type: 'FeatureCollection',
    features: [],
    meta: { districtDataVersionId: '', returnedCount: 0, bbox: [0, 0, 0, 0] },
  }
}

export function segmentQuery(
  bbox: [number, number, number, number],
  classifications: Classification[],
  filters: AppliedFilters,
): string {
  const query = new URLSearchParams({
    cityId: CITY_ID,
    bbox: bbox.map((coordinate) => coordinate.toFixed(6)).join(','),
    classification: classifications.join(','),
  })
  if (filters.minLength !== null) query.set('minLength', String(filters.minLength))
  if (filters.maxLength !== null) query.set('maxLength', String(filters.maxLength))
  return query.toString()
}

export function districtQuery(bbox: [number, number, number, number]): string {
  return new URLSearchParams({
    cityId: CITY_ID,
    bbox: bbox.map((coordinate) => coordinate.toFixed(6)).join(','),
  }).toString()
}

export function districtLabelCollection(collection: DistrictCollection): FeatureCollection<Point, DistrictProperties> {
  return {
    type: 'FeatureCollection',
    features: collection.features.map((feature) => ({
      type: 'Feature',
      id: feature.id,
      geometry: feature.properties.labelPoint,
      properties: feature.properties,
    })),
  }
}

export function endpointCollection(collection: SegmentCollection): FeatureCollection<Point> {
  const points: Feature<Point>[] = []
  const seen = new Set<string>()
  for (const feature of collection.features) {
    const coordinates = feature.geometry.coordinates
    for (const coordinate of [coordinates[0], coordinates.at(-1)]) {
      if (!coordinate) continue
      const key = `${coordinate[0].toFixed(7)},${coordinate[1].toFixed(7)},${feature.properties.boundaryClip}`
      if (seen.has(key)) continue
      seen.add(key)
      points.push({
        type: 'Feature',
        geometry: { type: 'Point', coordinates: coordinate },
        properties: { kind: feature.properties.boundaryClip ? 'boundary' : 'endpoint' },
      })
    }
  }
  return { type: 'FeatureCollection', features: points }
}

export function parseLength(value: string): number | null | 'invalid' {
  if (value.trim() === '') return null
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : 'invalid'
}
import type { Feature, FeatureCollection, LineString, MultiPolygon, Point, Polygon } from 'geojson'
