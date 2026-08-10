import { useEffect, useMemo, useRef, useState } from 'react'
import { GeoJSONSource, Map, NavigationControl, setWorkerUrl, type MapMouseEvent } from 'maplibre-gl'
import maplibreWorkerURL from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url'
import { isGeoPlaygroundPath } from './routing'
import {
  coverageCollection,
  matchedCollection,
  normalizedFeature,
  routeBounds,
  routeFeature,
  unmatchedCollection,
  type CoverageFeature,
  type RouteAnalysis,
  type SampleRoute,
  type SampleRouteCollection,
} from './routeAnalysis'
import {
  CITY_ID,
  CLASSIFICATIONS,
  EMPTY_STATISTICS,
  districtLabelCollection,
  districtQuery,
  emptyCollection,
  emptyDistrictCollection,
  endpointCollection,
  parseLength,
  segmentQuery,
  type APIErrorPayload,
  type AppliedFilters,
  type Classification,
  type DistrictCollection,
  type DistrictProperties,
  type GeoVersion,
  type SegmentCollection,
  type SegmentDetail,
} from './geo'

type ApiState = 'checking' | 'ready' | 'unavailable'
type Visibility = Record<Classification, boolean>

const apiURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'
const mapStyleURL =
  import.meta.env.VITE_MAP_STYLE_URL ?? 'https://tiles.openfreemap.org/styles/liberty'
const lineLayers: Record<Classification, string> = {
  EXPLORE: 'segments-explore',
  ROUTABLE_ONLY: 'segments-routable',
  IGNORE: 'segments-ignore',
}
const classificationLabels: Record<Classification, string> = {
  EXPLORE: 'Исследуемые',
  ROUTABLE_ONLY: 'Только связность',
  IGNORE: 'Исключённые',
}
const classificationColors: Record<Classification, string> = {
  EXPLORE: '#35d3b4',
  ROUTABLE_ONLY: '#f0b34d',
  IGNORE: '#b77774',
}
const coverageLayerIDs = ['coverage-not-covered', 'coverage-partial', 'coverage-completed', 'coverage-connector']

setWorkerUrl(maplibreWorkerURL)

export function App() {
  const mapContainer = useRef<HTMLDivElement>(null)
  const map = useRef<Map | null>(null)
  const baseLayerIDs = useRef<string[]>([])
  const requestViewportRef = useRef<() => void>(() => undefined)
  const viewportAbort = useRef<AbortController | null>(null)
  const detailAbort = useRef<AbortController | null>(null)
  const analysisAbort = useRef<AbortController | null>(null)
  const requestSequence = useRef(0)
  const [mapReady, setMapReady] = useState(false)
  const [apiState, setApiState] = useState<ApiState>('checking')
  const [version, setVersion] = useState<GeoVersion | null>(null)
  const [collection, setCollection] = useState<SegmentCollection>(emptyCollection)
  const [districtCollection, setDistrictCollection] = useState<DistrictCollection>(emptyDistrictCollection)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [visibility, setVisibility] = useState<Visibility>({
    EXPLORE: true,
    ROUTABLE_ONLY: true,
    IGNORE: true,
  })
  const visibilityRef = useRef(visibility)
  const showDistrictsRef = useRef(true)
  const [showBasemap, setShowBasemap] = useState(true)
  const [showPoints, setShowPoints] = useState(false)
  const [showDistricts, setShowDistricts] = useState(true)
  const [minimumInput, setMinimumInput] = useState('')
  const [maximumInput, setMaximumInput] = useState('')
  const [filterError, setFilterError] = useState<string | null>(null)
  const [filters, setFilters] = useState<AppliedFilters>({ minLength: null, maxLength: null })
  const filtersRef = useRef(filters)
  const [selectedID, setSelectedID] = useState<string | null>(null)
  const [selected, setSelected] = useState<SegmentDetail | null>(null)
  const [selectedDistrict, setSelectedDistrict] = useState<DistrictProperties | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [showDebug, setShowDebug] = useState(false)
  const [routes, setRoutes] = useState<SampleRoute[]>([])
  const [selectedRouteID, setSelectedRouteID] = useState('')
  const [coverageProfile, setCoverageProfile] = useState('balanced')
  const [customRadius, setCustomRadius] = useState('20')
  const [customRatio, setCustomRatio] = useState('0.6')
  const [customMinimum, setCustomMinimum] = useState('15')
  const [customMaximum, setCustomMaximum] = useState('80')
  const [analysis, setAnalysis] = useState<RouteAnalysis | null>(null)
  const [analysisLoading, setAnalysisLoading] = useState(false)
  const [analysisError, setAnalysisError] = useState<string | null>(null)
  const [selectedCoverage, setSelectedCoverage] = useState<CoverageFeature['properties'] | null>(null)
  const selectedRoute = useMemo(() => routes.find((route) => route.id === selectedRouteID) ?? null, [routes, selectedRouteID])

  useEffect(() => {
    visibilityRef.current = visibility
  }, [visibility])

  useEffect(() => {
    filtersRef.current = filters
  }, [filters])

  useEffect(() => {
    showDistrictsRef.current = showDistricts
  }, [showDistricts])

  useEffect(() => {
    if (!mapContainer.current || map.current) return
    const instance = new Map({
      container: mapContainer.current,
      style: mapStyleURL,
      center: [30.315, 59.9375],
      zoom: 14,
      minZoom: 13,
      maxZoom: 20,
      attributionControl: {},
    })
    map.current = instance
    instance.addControl(new NavigationControl(), 'bottom-right')

    const clearViewport = () => {
      setCollection(emptyCollection())
      setDistrictCollection(emptyDistrictCollection())
      setError(null)
      setLoading(false)
    }

    const requestViewport = () => {
      if (!instance.getSource('segments')) return
      if (instance.getZoom() < 13) {
        clearViewport()
        return
      }
      const classifications = CLASSIFICATIONS.filter((value) => visibilityRef.current[value])
      if (classifications.length === 0 && !showDistrictsRef.current) {
        viewportAbort.current?.abort()
        clearViewport()
        return
      }
      const bounds = instance.getBounds()
      const bbox: [number, number, number, number] = [
        bounds.getWest(), bounds.getSouth(), bounds.getEast(), bounds.getNorth(),
      ]
      viewportAbort.current?.abort()
      const controller = new AbortController()
      viewportAbort.current = controller
      const sequence = ++requestSequence.current
      setLoading(true)
      setError(null)
      const fetchJSON = async <T,>(url: string): Promise<T> => {
        const response = await fetch(url, { signal: controller.signal })
          if (!response.ok) {
            const payload = (await response.json().catch(() => ({}))) as APIErrorPayload
            throw new Error(payload.error?.message ?? `API вернул ${response.status}`)
          }
        return response.json() as Promise<T>
      }
      const segmentsRequest = classifications.length > 0
        ? fetchJSON<SegmentCollection>(`${apiURL}/api/v1/geo/segments?${segmentQuery(bbox, classifications, filtersRef.current)}`)
        : Promise.resolve(emptyCollection())
      const districtsRequest = showDistrictsRef.current
        ? fetchJSON<DistrictCollection>(`${apiURL}/api/v1/geo/districts?${districtQuery(bbox)}`)
        : Promise.resolve(emptyDistrictCollection())
      Promise.all([segmentsRequest, districtsRequest])
        .then(([segmentsResult, districtsResult]) => {
          if (sequence !== requestSequence.current) return
          setCollection(segmentsResult)
          setDistrictCollection(districtsResult)
          setLoading(false)
        })
        .catch((cause: unknown) => {
          if (controller.signal.aborted || sequence !== requestSequence.current) return
          setLoading(false)
          setError(cause instanceof Error ? cause.message : 'Не удалось загрузить геоданные')
        })
    }
    requestViewportRef.current = requestViewport

    let debounceTimer: number | undefined
    const onMoveEnd = () => {
      window.clearTimeout(debounceTimer)
      debounceTimer = window.setTimeout(requestViewport, 250)
    }
    const onMapClick = (event: MapMouseEvent) => {
      const coverageFeature = instance.queryRenderedFeatures(event.point, {
        layers: coverageLayerIDs.filter((layer) => instance.getLayer(layer)),
      })[0]
      if (coverageFeature?.properties?.id) {
        setSelectedCoverage(coverageFeature.properties as CoverageFeature['properties'])
        setSelectedID(null)
        setSelectedDistrict(null)
        return
      }
      const layers = Object.values(lineLayers).filter((layer) => instance.getLayer(layer))
      const feature = instance.queryRenderedFeatures(event.point, { layers })[0]
      const id = feature?.properties?.id
      if (typeof id === 'string') {
        setDetailLoading(true)
        setSelectedCoverage(null)
        setSelectedDistrict(null)
        setSelectedID(id)
        return
      }
      const district = instance.getLayer('district-fill')
        ? instance.queryRenderedFeatures(event.point, { layers: ['district-fill'] })[0]
        : undefined
      if (district?.properties && typeof district.properties.id === 'string') {
        setSelectedCoverage(null)
        setSelectedID(null)
        setSelected(null)
        setDetailLoading(false)
        setSelectedDistrict(district.properties as DistrictProperties)
      }
    }
    const onMouseMove = (event: MapMouseEvent) => {
      const layers = [...coverageLayerIDs, ...Object.values(lineLayers), 'district-fill'].filter((layer) => instance.getLayer(layer))
      instance.getCanvas().style.cursor = instance.queryRenderedFeatures(event.point, { layers }).length > 0 ? 'pointer' : ''
    }

    instance.once('style.load', () => {
      baseLayerIDs.current = (instance.getStyle().layers ?? []).map((layer) => layer.id)
      instance.addSource('districts', { type: 'geojson', data: emptyDistrictCollection() })
      instance.addSource('district-labels', { type: 'geojson', data: { type: 'FeatureCollection', features: [] } })
      instance.addLayer({
        id: 'district-fill', type: 'fill', source: 'districts',
        paint: { 'fill-color': '#7ba99b', 'fill-opacity': 0.11 },
      })
      instance.addLayer({
        id: 'district-outline', type: 'line', source: 'districts',
        paint: { 'line-color': '#a8cabe', 'line-width': ['interpolate', ['linear'], ['zoom'], 13, 1.1, 17, 2], 'line-opacity': 0.62 },
      })
      instance.addLayer({
        id: 'district-selection', type: 'line', source: 'districts',
        filter: ['==', ['get', 'id'], ''],
        paint: { 'line-color': '#ffffff', 'line-width': 3, 'line-opacity': 0.9 },
      })
      instance.addLayer({
        id: 'district-labels', type: 'symbol', source: 'district-labels',
        layout: { 'text-field': ['get', 'name'], 'text-size': 12, 'text-allow-overlap': false },
        paint: { 'text-color': '#d5e8e1', 'text-halo-color': '#142122', 'text-halo-width': 1.4, 'text-opacity': 0.82 },
      })
      instance.addSource('segments', { type: 'geojson', data: emptyCollection() })
      for (const classification of CLASSIFICATIONS) {
        instance.addLayer({
          id: lineLayers[classification],
          type: 'line',
          source: 'segments',
          filter: ['==', ['get', 'classification'], classification],
          paint: {
            'line-color': classificationColors[classification],
            'line-width': ['interpolate', ['linear'], ['zoom'], 13, 2, 17, 5],
            'line-opacity': classification === 'IGNORE' ? 0.72 : 0.9,
          },
        })
      }
      instance.addSource('route-source', { type: 'geojson', data: routeFeature(null) })
      instance.addLayer({
        id: 'route-source-line', type: 'line', source: 'route-source',
        paint: { 'line-color': '#b98cff', 'line-width': 3, 'line-dasharray': [2, 1.5], 'line-opacity': 0.8 },
      })
      instance.addSource('route-coverage', { type: 'geojson', data: coverageCollection(null) })
      for (const [id, status, color, opacity] of [
        ['coverage-not-covered', 'NOT_COVERED', '#82908d', 0.42],
        ['coverage-partial', 'PARTIAL', '#efc84b', 0.9],
        ['coverage-completed', 'COMPLETED', '#40d48f', 0.95],
        ['coverage-connector', 'CONNECTOR', '#55aef2', 0.95],
      ] as const) {
        instance.addLayer({
          id, type: 'line', source: 'route-coverage', filter: ['==', ['get', 'status'], status],
          paint: {
            'line-color': color, 'line-width': ['interpolate', ['linear'], ['zoom'], 13, 3, 17, 7],
            'line-opacity': opacity,
            ...(status === 'CONNECTOR' ? { 'line-dasharray': [1.5, 1.2] } : {}),
          },
        })
      }
      instance.addLayer({
        id: 'coverage-direct-outline', type: 'line', source: 'route-coverage',
        filter: ['in', ['get', 'provenance'], ['literal', ['DIRECT', 'DIRECT_AND_RADIUS']]],
        paint: { 'line-color': '#f5fff9', 'line-width': 1.2, 'line-opacity': 0.78, 'line-dasharray': [1, 1.6] },
      })
      instance.addSource('route-normalized', { type: 'geojson', data: normalizedFeature(null) })
      instance.addLayer({
        id: 'route-normalized-line', type: 'line', source: 'route-normalized',
        paint: { 'line-color': '#69f3d1', 'line-width': 2.2, 'line-opacity': 0.92 },
      })
      instance.addSource('route-matched', { type: 'geojson', data: matchedCollection(null) })
      instance.addLayer({
        id: 'route-matched-line', type: 'line', source: 'route-matched',
        paint: { 'line-color': '#d8fff4', 'line-width': 1.1, 'line-opacity': 0.72 },
      })
      instance.addSource('route-unmatched', { type: 'geojson', data: unmatchedCollection(null) })
      instance.addLayer({
        id: 'route-unmatched-line', type: 'line', source: 'route-unmatched',
        paint: { 'line-color': '#ff625f', 'line-width': 5, 'line-opacity': 0.98 },
      })
      instance.addLayer({
        id: 'segment-selection', type: 'line', source: 'segments',
        filter: ['==', ['get', 'id'], ''],
        paint: { 'line-color': '#ffffff', 'line-width': 7, 'line-opacity': 0.96 },
      })
      instance.addSource('segment-points', { type: 'geojson', data: { type: 'FeatureCollection', features: [] } })
      instance.addLayer({
        id: 'segment-endpoints', type: 'circle', source: 'segment-points',
        filter: ['==', ['get', 'kind'], 'endpoint'],
        paint: { 'circle-color': '#d8eee7', 'circle-radius': 2.2, 'circle-opacity': 0.55 },
      })
      instance.addLayer({
        id: 'segment-boundaries', type: 'circle', source: 'segment-points',
        filter: ['==', ['get', 'kind'], 'boundary'],
        paint: { 'circle-color': '#fa73ad', 'circle-radius': 4.5, 'circle-stroke-color': '#fff', 'circle-stroke-width': 1 },
      })
      setMapReady(true)
      instance.fitBounds([[30.3, 59.93], [30.33, 59.945]], { padding: 76, duration: 0, maxZoom: 15.2 })
      requestViewport()
    })
    instance.on('moveend', onMoveEnd)
    instance.on('click', onMapClick)
    instance.on('mousemove', onMouseMove)

    return () => {
      window.clearTimeout(debounceTimer)
      viewportAbort.current?.abort()
      detailAbort.current?.abort()
      analysisAbort.current?.abort()
      instance.remove()
      map.current = null
    }
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    fetch(`${apiURL}/api/v1/cities/${CITY_ID}/geo-version`, { signal: controller.signal })
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
    fetch(`${apiURL}/api/v1/geo/sample-routes?cityId=${CITY_ID}`, { signal: controller.signal })
      .then(async (response) => {
        if (!response.ok) throw new Error(`API вернул ${response.status}`)
        return response.json() as Promise<SampleRouteCollection>
      })
      .then((result) => {
        setRoutes(result.routes)
        setSelectedRouteID((current) => current || result.routes[0]?.id || '')
        if (result.warnings.length > 0) setAnalysisError(`Fixture warning: ${result.warnings.join(', ')}`)
      })
      .catch((cause: unknown) => {
        if (!controller.signal.aborted) setAnalysisError(cause instanceof Error ? cause.message : 'Маршруты недоступны')
      })
    return () => controller.abort()
  }, [])

  useEffect(() => {
    if (!mapReady || !map.current) return
    const source = map.current.getSource('segments') as GeoJSONSource
    source.setData(collection)
    const pointSource = map.current.getSource('segment-points') as GeoJSONSource
    pointSource.setData(endpointCollection(collection))
  }, [collection, mapReady])

  useEffect(() => {
    if (!mapReady || !map.current) return
    ;(map.current.getSource('districts') as GeoJSONSource).setData(districtCollection)
    ;(map.current.getSource('district-labels') as GeoJSONSource).setData(districtLabelCollection(districtCollection))
  }, [districtCollection, mapReady])

  useEffect(() => {
    if (!mapReady || !map.current) return
    ;(map.current.getSource('route-source') as GeoJSONSource).setData(routeFeature(selectedRoute))
    if (selectedRoute) map.current.fitBounds(routeBounds(selectedRoute), { padding: 110, duration: 550, maxZoom: 15.8 })
  }, [selectedRoute, mapReady])

  useEffect(() => {
    if (!mapReady || !map.current) return
    ;(map.current.getSource('route-normalized') as GeoJSONSource).setData(normalizedFeature(analysis))
    ;(map.current.getSource('route-matched') as GeoJSONSource).setData(matchedCollection(analysis))
    ;(map.current.getSource('route-unmatched') as GeoJSONSource).setData(unmatchedCollection(analysis))
    ;(map.current.getSource('route-coverage') as GeoJSONSource).setData(coverageCollection(analysis))
  }, [analysis, mapReady])

  useEffect(() => {
    if (!mapReady || !map.current) return
    for (const classification of CLASSIFICATIONS) {
      map.current.setLayoutProperty(lineLayers[classification], 'visibility', visibility[classification] ? 'visible' : 'none')
    }
    requestViewportRef.current()
  }, [visibility, mapReady])

  useEffect(() => {
    if (!mapReady || !map.current) return
    for (const layerID of baseLayerIDs.current) {
      if (map.current.getLayer(layerID)) map.current.setLayoutProperty(layerID, 'visibility', showBasemap ? 'visible' : 'none')
    }
  }, [showBasemap, mapReady])

  useEffect(() => {
    if (!mapReady || !map.current) return
    for (const layerID of ['segment-endpoints', 'segment-boundaries']) {
      map.current.setLayoutProperty(layerID, 'visibility', showPoints ? 'visible' : 'none')
    }
  }, [showPoints, mapReady])

  useEffect(() => {
    if (!mapReady || !map.current) return
    for (const layerID of ['district-fill', 'district-outline', 'district-selection', 'district-labels']) {
      map.current.setLayoutProperty(layerID, 'visibility', showDistricts ? 'visible' : 'none')
    }
    requestViewportRef.current()
  }, [showDistricts, mapReady])

  useEffect(() => {
    if (!mapReady || !map.current) return
    map.current.setFilter('segment-selection', ['==', ['get', 'id'], selectedID ?? ''])
  }, [selectedID, mapReady])

  useEffect(() => {
    if (!mapReady || !map.current) return
    map.current.setFilter('district-selection', ['==', ['get', 'id'], selectedDistrict?.id ?? ''])
  }, [selectedDistrict, mapReady])

  useEffect(() => {
    if (!selectedID) return
    detailAbort.current?.abort()
    const controller = new AbortController()
    detailAbort.current = controller
    fetch(`${apiURL}/api/v1/geo/segments/${selectedID}${showDebug ? '?debug=true' : ''}`, { signal: controller.signal })
      .then(async (response) => {
        if (!response.ok) throw new Error(`API вернул ${response.status}`)
        return response.json() as Promise<SegmentDetail>
      })
      .then((result) => {
        setSelected(result)
        setDetailLoading(false)
      })
      .catch(() => {
        if (!controller.signal.aborted) setDetailLoading(false)
      })
    return () => controller.abort()
  }, [selectedID, showDebug])

  const statistics = collection.meta?.statistics ?? EMPTY_STATISTICS
  const attributes = useMemo(() => selected ? Object.entries(selected.normalizedAttributes) : [], [selected])

  const applyFilters = () => {
    const minimum = parseLength(minimumInput)
    const maximum = parseLength(maximumInput)
    if (minimum === 'invalid' || maximum === 'invalid') {
      setFilterError('Длина должна быть неотрицательным числом')
      return
    }
    if (minimum !== null && maximum !== null && minimum > maximum) {
      setFilterError('Минимум не может быть больше максимума')
      return
    }
    setFilterError(null)
    const next = { minLength: minimum, maxLength: maximum }
    filtersRef.current = next
    setFilters(next)
    requestViewportRef.current()
  }

  const runAnalysis = () => {
    if (!selectedRoute) return
    const customValues = [customRadius, customRatio, customMinimum, customMaximum].map(Number)
    if (coverageProfile === 'custom' && customValues.some((value) => !Number.isFinite(value))) {
      setAnalysisError('Параметры custom-профиля должны быть числами')
      return
    }
    analysisAbort.current?.abort()
    const controller = new AbortController()
    analysisAbort.current = controller
    setAnalysisLoading(true)
    setAnalysisError(null)
    setSelectedCoverage(null)
    const analysisVisibility: Visibility = { EXPLORE: false, ROUTABLE_ONLY: false, IGNORE: false }
    visibilityRef.current = analysisVisibility
    setVisibility(analysisVisibility)
    const coverage = coverageProfile === 'custom'
      ? {
          profile: 'custom', radiusMeters: customValues[0], coverageRatio: customValues[1],
          minRequiredMeters: customValues[2], maxRequiredMeters: customValues[3],
        }
      : { profile: coverageProfile }
    fetch(`${apiURL}/api/v1/geo/sample-routes/${selectedRoute.id}/analyze?cityId=${CITY_ID}`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, signal: controller.signal,
      body: JSON.stringify({ coverage }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const payload = (await response.json().catch(() => ({}))) as APIErrorPayload
          throw new Error(payload.error?.message ?? `API вернул ${response.status}`)
        }
        return response.json() as Promise<RouteAnalysis>
      })
      .then((result) => { setAnalysis(result); setAnalysisLoading(false) })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return
        setAnalysisLoading(false)
        setAnalysisError(cause instanceof Error ? cause.message : 'Анализ не выполнен')
      })
  }

  if (!isGeoPlaygroundPath(window.location.pathname)) {
    return (
      <main className="not-found">
        <p className="eyebrow">ГуляЕм</p>
        <h1>Geo Playground</h1>
        <p>Инженерная карта доступна по адресу /debug/geo.</p>
        <a href="/debug/geo">Открыть playground</a>
      </main>
    )
  }

  return (
    <main className="playground">
      <div ref={mapContainer} className="map" aria-label="Карта сегментов центра Санкт-Петербурга" />
      <header className="topbar">
        <div>
          <p className="eyebrow">Stage 1.5 · Geo Playground</p>
          <h1>ГуляЕм <span>/ Matching & Coverage</span></h1>
        </div>
        <div className={`api-status api-status--${apiState}`} role="status">
          <span aria-hidden="true" />
          {apiState === 'checking' && 'Проверяем геоданные'}
          {apiState === 'ready' && `READY · ${version?.normalizationVersion ?? '—'}`}
          {apiState === 'unavailable' && 'API недоступен'}
        </div>
      </header>

      <aside className="control-panel panel">
        <div className="panel-heading">
          <div><p className="panel-label">Видимый viewport</p><h2>Слои и фильтры</h2></div>
          <span className={loading ? 'loading-dot loading-dot--active' : 'loading-dot'} aria-label={loading ? 'Загрузка' : 'Загружено'} />
        </div>
        <section className="route-controls">
          <p className="panel-label">Sample route</p>
          <select value={selectedRouteID} onChange={(event) => {
            setSelectedRouteID(event.target.value)
            setAnalysis(null)
            setSelectedCoverage(null)
          }} aria-label="Тестовый маршрут">
            {routes.map((route) => <option key={route.id} value={route.id}>{route.name}</option>)}
          </select>
          {selectedRoute && <p className="route-description">{selectedRoute.description}{selectedRoute.intentionalUnmatched ? ' · есть намеренный unmatched-фрагмент' : ''}</p>}
          <div className="profile-row">
            <label>Профиль<select value={coverageProfile} onChange={(event) => setCoverageProfile(event.target.value)}>
              <option value="strict">Strict · 10 м</option>
              <option value="balanced">Balanced · 20 м</option>
              <option value="generous">Generous · 35 м</option>
              <option value="custom">Custom</option>
            </select></label>
            <button onClick={runAnalysis} disabled={!selectedRoute || analysisLoading}>{analysisLoading ? 'Считаем…' : 'Анализировать'}</button>
          </div>
          {coverageProfile === 'custom' && <div className="custom-profile">
            <label>Радиус, м<input value={customRadius} onChange={(event) => setCustomRadius(event.target.value)} inputMode="decimal" /></label>
            <label>Доля<input value={customRatio} onChange={(event) => setCustomRatio(event.target.value)} inputMode="decimal" /></label>
            <label>Min, м<input value={customMinimum} onChange={(event) => setCustomMinimum(event.target.value)} inputMode="decimal" /></label>
            <label>Max, м<input value={customMaximum} onChange={(event) => setCustomMaximum(event.target.value)} inputMode="decimal" /></label>
          </div>}
          {analysisError && <p className="inline-error">{analysisError}</p>}
          {analysis && <>
            <div className="coverage-legend">
              <span className="completed">Completed</span><span className="partial">Partial</span>
              <span className="not-covered">Not covered</span><span className="connector">Connector</span>
            </div>
            <div className="stats-grid route-stats">
              <Stat label="Matched" value={`${formatNumber(analysis.metrics.routeMatchedRatio * 100)}%`} />
              <Stat label="Completed" value={`${formatNumber(analysis.metrics.completedNetworkRatio * 100)}%`} />
              <Stat label="Покрыто" value={formatDistance(analysis.metrics.geometricCoveredLengthMeters)} />
              <Stat label="Unmatched" value={formatDistance(analysis.metrics.routeUnmatchedLengthMeters)} />
            </div>
          </>}
        </section>
        <div className="classification-list">
          {CLASSIFICATIONS.map((classification) => (
            <label className="layer-toggle" key={classification}>
              <input type="checkbox" checked={visibility[classification]} onChange={() => setVisibility((current) => ({ ...current, [classification]: !current[classification] }))} />
              <i style={{ backgroundColor: classificationColors[classification] }} />
              <span>{classificationLabels[classification]}</span>
              <b>{classification === 'EXPLORE' ? statistics.exploreCount : classification === 'ROUTABLE_ONLY' ? statistics.routableOnlyCount : statistics.ignoreCount}</b>
            </label>
          ))}
        </div>
        <div className="secondary-toggles">
          <label><input type="checkbox" checked={showBasemap} onChange={(event) => setShowBasemap(event.target.checked)} /> Подложка</label>
          <label><input type="checkbox" checked={showDistricts} onChange={(event) => { setShowDistricts(event.target.checked); if (!event.target.checked) setSelectedDistrict(null) }} /> Районы</label>
          <label><input type="checkbox" checked={showPoints} onChange={(event) => setShowPoints(event.target.checked)} /> Узлы сегментов</label>
        </div>
        <div className="length-filter">
          <p className="panel-label">Длина, м</p>
          <div><input inputMode="decimal" value={minimumInput} onChange={(event) => setMinimumInput(event.target.value)} placeholder="от" aria-label="Минимальная длина" /><span>—</span><input inputMode="decimal" value={maximumInput} onChange={(event) => setMaximumInput(event.target.value)} placeholder="до" aria-label="Максимальная длина" /><button onClick={applyFilters}>Применить</button></div>
          {filterError && <p className="inline-error">{filterError}</p>}
        </div>
        {error && <div className="map-error"><strong>Viewport не загружен</strong><span>{error}</span></div>}
        <div className="stats-grid">
          <Stat label="Сегменты" value={formatInteger(statistics.segmentsTotal)} />
          <Stat label="Всего" value={formatDistance(statistics.totalLengthMeters)} />
          <Stat label="Медиана" value={`${formatNumber(statistics.medianLengthMeters)} м`} />
          <Stat label="P95" value={`${formatNumber(statistics.p95LengthMeters)} м`} />
        </div>
        <div className="diagnostics"><span>&lt; 5 м <b>{statistics.shortSegmentCount}</b></span><span>&gt; 500 м <b>{statistics.longSegmentCount}</b></span></div>
      </aside>

      <aside className={`inspector panel ${selectedID || selectedDistrict || selectedCoverage ? 'inspector--open' : ''}`} aria-live="polite">
        <div className="inspector-handle" />
        <div className="panel-heading">
          <div><p className="panel-label">Инспектор</p><h2>{selectedCoverage ? 'Покрытие сегмента' : selectedDistrict?.name ?? selected?.street?.name ?? (selectedID ? 'Сегмент' : 'Выберите объект')}</h2></div>
          {(selectedID || selectedDistrict || selectedCoverage) && <button className="icon-button" onClick={() => { setSelectedID(null); setSelected(null); setSelectedDistrict(null); setSelectedCoverage(null) }} aria-label="Закрыть инспектор">×</button>}
        </div>
        {!selectedID && !selectedDistrict && !selectedCoverage && <p className="empty-message">Нажмите на цветную линию, покрытие или район, чтобы увидеть детали.</p>}
        {detailLoading && <p className="empty-message">Загружаем детали…</p>}
        {selectedDistrict && <>
          <div className="district-summary"><span>Административный район</span></div>
          <dl className="detail-list">
            <Detail label="Тип" value={selectedDistrict.kind} />
            <Detail label="Источник" value={selectedDistrict.source} />
            <Detail label="Версия" value={selectedDistrict.districtDataVersionId} code />
            <Detail label="Нормализация" value={selectedDistrict.normalizationVersion} />
            <Detail label="External ID" value={selectedDistrict.externalId} code />
          </dl>
        </>}
        {selectedCoverage && <>
          <div className="segment-summary">
            <span>{selectedCoverage.status}</span>
            <strong>{formatDistance(Number(selectedCoverage.coveredMeters))}</strong>
          </div>
          <dl className="detail-list">
            <Detail label="Происхождение" value={selectedCoverage.provenance || '—'} />
            <Detail label="Требуется" value={formatDistance(Number(selectedCoverage.requiredMeters))} />
            <Detail label="Длина" value={formatDistance(Number(selectedCoverage.lengthMeters))} />
            <Detail label="ID" value={selectedCoverage.id} code />
          </dl>
        </>}
        {selected && !detailLoading && <>
          <div className="segment-summary">
            <span style={{ color: classificationColors[selected.classification] }}>{classificationLabels[selected.classification]}</span>
            <strong>{formatNumber(selected.lengthMeters)} м</strong>
          </div>
          <dl className="detail-list">
            <Detail label="Причина" value={selected.reasonCode} />
            <Detail label="Версия" value={`${selected.versionStatus}${selected.isCurrent ? ' · current' : ''}`} />
            <Detail label="Нормализация" value={selected.normalizationVersion} />
            <Detail label="ID" value={selected.id} code />
          </dl>
          <p className="panel-label attributes-title">Районы</p>
          {selected.districts.length === 0 ? <p className="empty-message compact">Район не определён</p> : <div className="district-chips">{selected.districts.map((district) => <span key={district.id}>{district.name}</span>)}</div>}
          <p className="panel-label attributes-title">Нормализованные атрибуты</p>
          {attributes.length === 0 ? <p className="empty-message compact">Нет дополнительных атрибутов</p> : <dl className="attribute-list">{attributes.map(([key, value]) => <Detail key={key} label={key} value={formatAttribute(value)} code />)}</dl>}
          <label className="debug-toggle"><input type="checkbox" checked={showDebug} onChange={(event) => { setDetailLoading(true); setShowDebug(event.target.checked) }} /> Показать OSM debug metadata</label>
          {showDebug && selected.debugSource && <pre className="debug-json">{JSON.stringify(selected.debugSource, null, 2)}</pre>}
        </>}
      </aside>
    </main>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return <div><span>{label}</span><strong>{value}</strong></div>
}

function Detail({ label, value, code = false }: { label: string; value: string; code?: boolean }) {
  return <div><dt>{label}</dt><dd className={code ? 'code-value' : ''}>{value}</dd></div>
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 1 }).format(value)
}

function formatInteger(value: number): string {
  return new Intl.NumberFormat('ru-RU').format(value)
}

function formatDistance(meters: number): string {
  return meters >= 1000 ? `${formatNumber(meters / 1000)} км` : `${formatNumber(meters)} м`
}

function formatAttribute(value: unknown): string {
  return typeof value === 'string' ? value : JSON.stringify(value)
}
