import { useEffect, useState } from 'react'
import type { FeatureCollection, LineString } from 'geojson'
import { API_URL } from '../geoPlayground/config'
import { CITY_ID } from '../geo'
import type { MapViewport } from '../geoPlayground/useViewportData'
import type { ExplorationSummary } from './types'

const EMPTY: FeatureCollection<LineString> = { type: 'FeatureCollection', features: [] }

export function useExploration(viewport: MapViewport | null, refreshToken: number) {
  const [summary, setSummary] = useState<ExplorationSummary | null>(null)
  const [segments, setSegments] = useState<FeatureCollection<LineString>>(EMPTY)
  const [error, setError] = useState<string | null>(null)
  useEffect(() => { fetch(`${API_URL}/api/v1/cities/${CITY_ID}/exploration`).then(async response => { if (!response.ok) throw await response.json(); return response.json() as Promise<ExplorationSummary> }).then(value => { setSummary(value); setError(null) }).catch((cause: { code?: string }) => setError(cause.code === 'exploration_rebuild_required' ? 'Исследованную карту нужно перестроить после обновления геоданных.' : 'Не удалось загрузить прогресс.')) }, [refreshToken])
  useEffect(() => {
    if (!viewport) return
    const controller = new AbortController(); const bbox = viewport.bbox.join(',')
    fetch(`${API_URL}/api/v1/cities/${CITY_ID}/exploration/segments?bbox=${bbox}`, { signal: controller.signal }).then(response => { if (!response.ok) throw new Error(); return response.json() as Promise<FeatureCollection<LineString>> }).then(setSegments).catch(() => { if (!controller.signal.aborted) setSegments(EMPTY) })
    return () => controller.abort()
  }, [viewport, refreshToken])
  return { summary, segments, error }
}
