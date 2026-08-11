import { useEffect, useRef, useState } from 'react'
import { GeoJSONSource, Map, NavigationControl, setWorkerUrl, type MapMouseEvent } from 'maplibre-gl'
import maplibreWorkerURL from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url'
import {
  CLASSIFICATIONS,
  districtLabelCollection,
  emptyCollection,
  emptyDistrictCollection,
  endpointCollection,
  type DistrictCollection,
  type DistrictProperties,
  type SegmentCollection,
} from '../geo'
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
} from '../routeAnalysis'
import {
  ROUTING_ENGINES,
  comparisonCollection,
  waypointCollection,
  type RoutingComparison,
  type RoutingEngineID,
} from '../routingComparison'
import {
  CLASSIFICATION_COLORS,
  COVERAGE_LAYER_IDS,
  LINE_LAYERS,
  MAP_STYLE_URL,
  ROUTING_COLORS,
  ROUTING_LAYER_IDS,
  type Visibility,
} from './config'
import type { MapViewport } from './useViewportData'

setWorkerUrl(maplibreWorkerURL)

interface GeoMapProps {
  collection: SegmentCollection
  districtCollection: DistrictCollection
  selectedRoute: SampleRoute | null
  analysis: RouteAnalysis | null
  routingComparison: RoutingComparison | null
  selectedRouteID: string
  visibility: Visibility
  showBasemap: boolean
  showPoints: boolean
  showDistricts: boolean
  showRoutingComparison: boolean
  routingVisibility: Record<RoutingEngineID, boolean>
  selectedSegmentID: string | null
  selectedDistrictID: string | null
  onViewportChange: (viewport: MapViewport) => void
  onSelectSegment: (id: string) => void
  onSelectDistrict: (district: DistrictProperties) => void
  onSelectCoverage: (coverage: CoverageFeature['properties']) => void
}

export function GeoMap(props: GeoMapProps) {
  const container = useRef<HTMLDivElement>(null)
  const map = useRef<Map | null>(null)
  const baseLayerIDs = useRef<string[]>([])
  const callbacks = useRef({
    onViewportChange: props.onViewportChange,
    onSelectSegment: props.onSelectSegment,
    onSelectDistrict: props.onSelectDistrict,
    onSelectCoverage: props.onSelectCoverage,
  })
  const [ready, setReady] = useState(false)

  useEffect(() => {
    callbacks.current = {
      onViewportChange: props.onViewportChange,
      onSelectSegment: props.onSelectSegment,
      onSelectDistrict: props.onSelectDistrict,
      onSelectCoverage: props.onSelectCoverage,
    }
  }, [props.onViewportChange, props.onSelectSegment, props.onSelectDistrict, props.onSelectCoverage])

  useEffect(() => {
    if (!container.current || map.current) return
    const instance = new Map({
      container: container.current,
      style: MAP_STYLE_URL,
      center: [30.315, 59.9375],
      zoom: 14,
      minZoom: 13,
      maxZoom: 20,
      attributionControl: {},
    })
    map.current = instance
    if (import.meta.env.DEV) window.__GULYAEM_DEBUG_MAP__ = instance
    instance.addControl(new NavigationControl(), 'bottom-right')

    const publishViewport = () => {
      if (!instance.getSource('segments')) return
      const bounds = instance.getBounds()
      callbacks.current.onViewportChange({
        bbox: [bounds.getWest(), bounds.getSouth(), bounds.getEast(), bounds.getNorth()],
        zoom: instance.getZoom(),
      })
    }
    const onMapClick = (event: MapMouseEvent) => {
      const coverage = instance.queryRenderedFeatures(event.point, {
        layers: COVERAGE_LAYER_IDS.filter((layer) => instance.getLayer(layer)),
      })[0]
      if (coverage?.properties?.id) {
        callbacks.current.onSelectCoverage(coverage.properties as CoverageFeature['properties'])
        return
      }
      const segment = instance.queryRenderedFeatures(event.point, {
        layers: Object.values(LINE_LAYERS).filter((layer) => instance.getLayer(layer)),
      })[0]
      if (typeof segment?.properties?.id === 'string') {
        callbacks.current.onSelectSegment(segment.properties.id)
        return
      }
      const district = instance.getLayer('district-fill')
        ? instance.queryRenderedFeatures(event.point, { layers: ['district-fill'] })[0]
        : undefined
      if (district?.properties && typeof district.properties.id === 'string') {
        callbacks.current.onSelectDistrict(district.properties as DistrictProperties)
      }
    }
    const onMouseMove = (event: MapMouseEvent) => {
      const layers = [...COVERAGE_LAYER_IDS, ...Object.values(LINE_LAYERS), 'district-fill']
        .filter((layer) => instance.getLayer(layer))
      instance.getCanvas().style.cursor = instance.queryRenderedFeatures(event.point, { layers }).length > 0
        ? 'pointer'
        : ''
    }

    instance.once('style.load', () => {
      baseLayerIDs.current = (instance.getStyle().layers ?? []).map((layer) => layer.id)
      addSourcesAndLayers(instance)
      setReady(true)
      instance.fitBounds([[30.3, 59.93], [30.33, 59.945]], { padding: 76, duration: 0, maxZoom: 15.2 })
      publishViewport()
    })
    instance.on('moveend', publishViewport)
    instance.on('click', onMapClick)
    instance.on('mousemove', onMouseMove)

    return () => {
      if (window.__GULYAEM_DEBUG_MAP__ === instance) delete window.__GULYAEM_DEBUG_MAP__
      instance.remove()
      map.current = null
    }
  }, [])

  useEffect(() => {
    if (!ready || !map.current) return
    ;(map.current.getSource('segments') as GeoJSONSource).setData(props.collection)
    ;(map.current.getSource('segment-points') as GeoJSONSource).setData(endpointCollection(props.collection))
  }, [props.collection, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    ;(map.current.getSource('districts') as GeoJSONSource).setData(props.districtCollection)
    ;(map.current.getSource('district-labels') as GeoJSONSource)
      .setData(districtLabelCollection(props.districtCollection))
  }, [props.districtCollection, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    ;(map.current.getSource('route-source') as GeoJSONSource).setData(routeFeature(props.selectedRoute))
    if (props.selectedRoute) {
      map.current.fitBounds(routeBounds(props.selectedRoute), { padding: 110, duration: 550, maxZoom: 15.8 })
    }
  }, [props.selectedRoute, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    ;(map.current.getSource('route-normalized') as GeoJSONSource).setData(normalizedFeature(props.analysis))
    ;(map.current.getSource('route-matched') as GeoJSONSource).setData(matchedCollection(props.analysis))
    ;(map.current.getSource('route-unmatched') as GeoJSONSource).setData(unmatchedCollection(props.analysis))
    ;(map.current.getSource('route-coverage') as GeoJSONSource).setData(coverageCollection(props.analysis))
  }, [props.analysis, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    ;(map.current.getSource('routing-comparison') as GeoJSONSource)
      .setData(comparisonCollection(props.routingComparison, props.selectedRouteID))
    ;(map.current.getSource('routing-waypoints') as GeoJSONSource)
      .setData(waypointCollection(props.routingComparison, props.selectedRouteID))
  }, [props.routingComparison, props.selectedRouteID, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    for (const engine of ROUTING_ENGINES) {
      map.current.setLayoutProperty(
        ROUTING_LAYER_IDS[engine],
        'visibility',
        props.showRoutingComparison && props.routingVisibility[engine] ? 'visible' : 'none',
      )
    }
    map.current.setLayoutProperty(
      'routing-waypoints', 'visibility', props.showRoutingComparison ? 'visible' : 'none',
    )
  }, [props.showRoutingComparison, props.routingVisibility, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    for (const classification of CLASSIFICATIONS) {
      map.current.setLayoutProperty(
        LINE_LAYERS[classification], 'visibility', props.visibility[classification] ? 'visible' : 'none',
      )
    }
  }, [props.visibility, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    for (const layerID of baseLayerIDs.current) {
      if (map.current.getLayer(layerID)) {
        map.current.setLayoutProperty(layerID, 'visibility', props.showBasemap ? 'visible' : 'none')
      }
    }
  }, [props.showBasemap, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    for (const layerID of ['segment-endpoints', 'segment-boundaries']) {
      map.current.setLayoutProperty(layerID, 'visibility', props.showPoints ? 'visible' : 'none')
    }
  }, [props.showPoints, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    for (const layerID of ['district-fill', 'district-outline', 'district-selection', 'district-labels']) {
      map.current.setLayoutProperty(layerID, 'visibility', props.showDistricts ? 'visible' : 'none')
    }
  }, [props.showDistricts, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    map.current.setFilter('segment-selection', ['==', ['get', 'id'], props.selectedSegmentID ?? ''])
  }, [props.selectedSegmentID, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    map.current.setFilter('district-selection', ['==', ['get', 'id'], props.selectedDistrictID ?? ''])
  }, [props.selectedDistrictID, ready])

  return <div ref={container} className="map" aria-label="Карта сегментов центра Санкт-Петербурга" />
}

function addSourcesAndLayers(instance: Map) {
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
    id: 'district-selection', type: 'line', source: 'districts', filter: ['==', ['get', 'id'], ''],
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
      id: LINE_LAYERS[classification], type: 'line', source: 'segments',
      filter: ['==', ['get', 'classification'], classification],
      paint: {
        'line-color': CLASSIFICATION_COLORS[classification],
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
  instance.addSource('routing-comparison', { type: 'geojson', data: comparisonCollection(null, '') })
  for (const engine of ROUTING_ENGINES) {
    instance.addLayer({
      id: ROUTING_LAYER_IDS[engine], type: 'line', source: 'routing-comparison',
      filter: ['==', ['get', 'engineId'], engine], layout: { visibility: 'none' },
      paint: {
        'line-color': ROUTING_COLORS[engine],
        'line-width': ['interpolate', ['linear'], ['zoom'], 13, 3, 17, 6],
        'line-opacity': 0.9,
      },
    })
  }
  instance.addSource('routing-waypoints', { type: 'geojson', data: waypointCollection(null, '') })
  instance.addLayer({
    id: 'routing-waypoints', type: 'circle', source: 'routing-waypoints', layout: { visibility: 'none' },
    paint: { 'circle-color': '#f6fff9', 'circle-radius': 5, 'circle-stroke-color': '#172526', 'circle-stroke-width': 2 },
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
        'line-color': color,
        'line-width': ['interpolate', ['linear'], ['zoom'], 13, 3, 17, 7],
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
    id: 'segment-selection', type: 'line', source: 'segments', filter: ['==', ['get', 'id'], ''],
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
}
