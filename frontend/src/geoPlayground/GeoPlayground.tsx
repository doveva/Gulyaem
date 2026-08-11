import { useEffect, useMemo, useRef, useState } from 'react'
import {
  CITY_ID,
  type APIErrorPayload,
  type AppliedFilters,
  type GeoVersion,
} from '../geo'
import type { RouteAnalysis, SampleRoute, SampleRouteCollection } from '../routeAnalysis'
import type { RoutingComparison, RoutingEngineID } from '../routingComparison'
import { API_URL, type Visibility } from './config'
import { GeoMap } from './GeoMap'
import { LayerControls } from './LayerControls'
import { RouteAnalysisPanel, type CoverageRequest } from './RouteAnalysisPanel'
import { RoutingComparisonPanel } from './RoutingComparisonPanel'
import { SegmentInspector } from './SegmentInspector'
import type { PlaygroundSelection } from './types'
import { useViewportData, type MapViewport } from './useViewportData'

type ApiState = 'checking' | 'ready' | 'unavailable'

export function GeoPlayground() {
  const analysisAbort = useRef<AbortController | null>(null)
  const [apiState, setApiState] = useState<ApiState>('checking')
  const [version, setVersion] = useState<GeoVersion | null>(null)
  const [viewport, setViewport] = useState<MapViewport | null>(null)
  const [visibility, setVisibility] = useState<Visibility>({
    EXPLORE: true,
    ROUTABLE_ONLY: true,
    IGNORE: true,
  })
  const [showBasemap, setShowBasemap] = useState(true)
  const [showPoints, setShowPoints] = useState(false)
  const [showDistricts, setShowDistricts] = useState(true)
  const [filters, setFilters] = useState<AppliedFilters>({ minLength: null, maxLength: null })
  const [selection, setSelection] = useState<PlaygroundSelection>(null)
  const [routes, setRoutes] = useState<SampleRoute[]>([])
  const [selectedRouteID, setSelectedRouteID] = useState('')
  const [analysis, setAnalysis] = useState<RouteAnalysis | null>(null)
  const [analysisLoading, setAnalysisLoading] = useState(false)
  const [analysisError, setAnalysisError] = useState<string | null>(null)
  const [routingComparison, setRoutingComparison] = useState<RoutingComparison | null>(null)
  const [routingComparisonError, setRoutingComparisonError] = useState<string | null>(null)
  const [showRoutingComparison, setShowRoutingComparison] = useState(false)
  const [routingVisibility, setRoutingVisibility] = useState<Record<RoutingEngineID, boolean>>({
    valhalla: true,
    graphhopper: true,
    osrm: true,
  })
  const viewportData = useViewportData(viewport, visibility, filters, showDistricts)
  const selectedRoute = useMemo(
    () => routes.find((route) => route.id === selectedRouteID) ?? null,
    [routes, selectedRouteID],
  )

  useEffect(() => {
    const controller = new AbortController()
    fetch(`${API_URL}/api/v1/cities/${CITY_ID}/geo-version`, { signal: controller.signal })
      .then(async (response) => {
        if (!response.ok) throw new Error(`API вернул ${response.status}`)
        return response.json() as Promise<GeoVersion>
      })
      .then((result) => {
        setVersion(result)
        setApiState('ready')
      })
      .catch(() => {
        if (!controller.signal.aborted) setApiState('unavailable')
      })
    return () => controller.abort()
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    fetch('/routing-spike/comparison.json', { signal: controller.signal })
      .then(async (response) => {
        if (!response.ok) throw new Error(`Отчёт Stage 1.6 недоступен (${response.status})`)
        return response.json() as Promise<RoutingComparison>
      })
      .then((result) => {
        setRoutingComparison(result)
        setRoutingComparisonError(null)
      })
      .catch((cause: unknown) => {
        if (!controller.signal.aborted) {
          setRoutingComparisonError(cause instanceof Error ? cause.message : 'Отчёт Stage 1.6 недоступен')
        }
      })
    return () => controller.abort()
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    fetch(`${API_URL}/api/v1/geo/sample-routes?cityId=${CITY_ID}`, { signal: controller.signal })
      .then(async (response) => {
        if (!response.ok) throw new Error(`API вернул ${response.status}`)
        return response.json() as Promise<SampleRouteCollection>
      })
      .then((result) => {
        setRoutes(result.routes)
        setSelectedRouteID((current) => current || result.routes[0]?.id || '')
        if (result.warnings.length > 0) {
          setAnalysisError(`Fixture warning: ${result.warnings.join(', ')}`)
        }
      })
      .catch((cause: unknown) => {
        if (!controller.signal.aborted) {
          setAnalysisError(cause instanceof Error ? cause.message : 'Маршруты недоступны')
        }
      })
    return () => controller.abort()
  }, [])

  useEffect(() => () => analysisAbort.current?.abort(), [])

  const changeRoute = (routeID: string) => {
    analysisAbort.current?.abort()
    setAnalysisLoading(false)
    setSelectedRouteID(routeID)
    setAnalysis(null)
    setSelection((current) => current?.kind === 'coverage' ? null : current)
  }

  const runAnalysis = (coverage: CoverageRequest) => {
    if (!selectedRoute) return
    analysisAbort.current?.abort()
    const controller = new AbortController()
    analysisAbort.current = controller
    setAnalysisLoading(true)
    setAnalysisError(null)
    setSelection((current) => current?.kind === 'coverage' ? null : current)
    setVisibility({ EXPLORE: false, ROUTABLE_ONLY: false, IGNORE: false })
    fetch(`${API_URL}/api/v1/geo/sample-routes/${selectedRoute.id}/analyze?cityId=${CITY_ID}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      signal: controller.signal,
      body: JSON.stringify({ coverage }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const payload = (await response.json().catch(() => ({}))) as APIErrorPayload
          throw new Error(payload.error?.message ?? `API вернул ${response.status}`)
        }
        return response.json() as Promise<RouteAnalysis>
      })
      .then((result) => {
        setAnalysis(result)
        setAnalysisLoading(false)
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return
        setAnalysisLoading(false)
        setAnalysisError(cause instanceof Error ? cause.message : 'Анализ не выполнен')
      })
  }

  const changeDistrictVisibility = (value: boolean) => {
    setShowDistricts(value)
    if (!value) setSelection((current) => current?.kind === 'district' ? null : current)
  }

  const selectedSegmentID = selection?.kind === 'segment' ? selection.id : null
  const selectedDistrictID = selection?.kind === 'district' ? selection.district.id : null

  return <main className="playground">
    <GeoMap
      collection={viewportData.collection}
      districtCollection={viewportData.districtCollection}
      selectedRoute={selectedRoute}
      analysis={analysis}
      routingComparison={routingComparison}
      selectedRouteID={selectedRouteID}
      visibility={visibility}
      showBasemap={showBasemap}
      showPoints={showPoints}
      showDistricts={showDistricts}
      showRoutingComparison={showRoutingComparison}
      routingVisibility={routingVisibility}
      selectedSegmentID={selectedSegmentID}
      selectedDistrictID={selectedDistrictID}
      onViewportChange={setViewport}
      onSelectSegment={(id) => setSelection({ kind: 'segment', id })}
      onSelectDistrict={(district) => setSelection({ kind: 'district', district })}
      onSelectCoverage={(coverage) => setSelection({ kind: 'coverage', coverage })}
    />
    <header className="topbar">
      <div>
        <p className="eyebrow">Stage 1.7 · Geo Playground</p>
        <h1>ГуляЕм <span>/ Validation &amp; freeze</span></h1>
      </div>
      <div className={`api-status api-status--${apiState}`} role="status">
        <span aria-hidden="true" />
        {apiState === 'checking' && 'Проверяем геоданные'}
        {apiState === 'ready' && `READY · ${version?.normalizationVersion ?? '—'}`}
        {apiState === 'unavailable' && 'API недоступен'}
      </div>
    </header>
    <LayerControls
      loading={viewportData.loading}
      error={viewportData.error}
      statistics={viewportData.collection.meta?.statistics}
      visibility={visibility}
      showBasemap={showBasemap}
      showPoints={showPoints}
      showDistricts={showDistricts}
      onVisibilityChange={setVisibility}
      onShowBasemapChange={setShowBasemap}
      onShowPointsChange={setShowPoints}
      onShowDistrictsChange={changeDistrictVisibility}
      onFiltersChange={setFilters}
    >
      <RouteAnalysisPanel
        routes={routes}
        selectedRoute={selectedRoute}
        selectedRouteID={selectedRouteID}
        analysis={analysis}
        loading={analysisLoading}
        error={analysisError}
        onRouteChange={changeRoute}
        onAnalyze={runAnalysis}
      />
      <RoutingComparisonPanel
        comparison={routingComparison}
        selectedRouteID={selectedRouteID}
        error={routingComparisonError}
        visible={showRoutingComparison}
        engineVisibility={routingVisibility}
        onVisibleChange={setShowRoutingComparison}
        onEngineVisibilityChange={setRoutingVisibility}
      />
    </LayerControls>
    <SegmentInspector selection={selection} onClose={() => setSelection(null)} />
  </main>
}
