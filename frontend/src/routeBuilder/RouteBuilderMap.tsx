import { useEffect, useRef, useState } from 'react'
import { GeoJSONSource, Map as MapLibreMap, Marker, NavigationControl, setWorkerUrl, type MapMouseEvent } from 'maplibre-gl'
import maplibreWorkerURL from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url'
import type { FeatureCollection, LineString } from 'geojson'
import { emptyCollection, type SegmentCollection } from '../geo'
import { MAP_STYLE_URL } from '../geoPlayground/config'
import type { MapViewport } from '../geoPlayground/useViewportData'
import type { RoutePreview, Waypoint } from './types'
import { waypointLabel } from './state'

setWorkerUrl(maplibreWorkerURL)

interface RouteBuilderMapProps {
  collection: SegmentCollection
  waypoints: Waypoint[]
  preview: RoutePreview | null
  routeGeometry?: LineString
  explored: FeatureCollection<LineString>
  newlyExplored?: FeatureCollection<LineString>
  calculating: boolean
  acceptingPoint: boolean
  onMapPoint: (lat: number, lon: number) => void
  onMoveWaypoint: (id: string, lat: number, lon: number) => void
  onViewportChange: (viewport: MapViewport) => void
}

export function RouteBuilderMap(props: RouteBuilderMapProps) {
  const container = useRef<HTMLDivElement>(null)
  const map = useRef<MapLibreMap | null>(null)
  const markers = useRef<Map<string, Marker>>(new globalThis.Map())
  const callbacks = useRef({
    onMapPoint: props.onMapPoint,
    onMoveWaypoint: props.onMoveWaypoint,
    onViewportChange: props.onViewportChange,
  })
  const [ready, setReady] = useState(false)

  useEffect(() => {
    callbacks.current = {
      onMapPoint: props.onMapPoint,
      onMoveWaypoint: props.onMoveWaypoint,
      onViewportChange: props.onViewportChange,
    }
  }, [props.onMapPoint, props.onMoveWaypoint, props.onViewportChange])

  useEffect(() => {
    if (!container.current || map.current) return
    const instance = new MapLibreMap({
      container: container.current,
      style: MAP_STYLE_URL,
      center: [30.315, 59.9375],
      zoom: 14.4,
      minZoom: 12,
      maxZoom: 20,
      attributionControl: {},
    })
    map.current = instance
    const markerCollection = markers.current
    if (import.meta.env.DEV) window.__GULYAEM_DEBUG_MAP__ = instance
    instance.addControl(new NavigationControl(), 'bottom-right')
    const publishViewport = () => {
      if (!instance.getSource('product-segments')) return
      const bounds = instance.getBounds()
      callbacks.current.onViewportChange({
        bbox: [bounds.getWest(), bounds.getSouth(), bounds.getEast(), bounds.getNorth()],
        zoom: instance.getZoom(),
      })
    }
    instance.once('style.load', () => {
      addProductLayers(instance)
      setReady(true)
      publishViewport()
    })
    instance.on('moveend', publishViewport)
    instance.on('click', (event: MapMouseEvent) => callbacks.current.onMapPoint(event.lngLat.lat, event.lngLat.lng))
    return () => {
      for (const marker of markerCollection.values()) marker.remove()
      markerCollection.clear()
      if (window.__GULYAEM_DEBUG_MAP__ === instance) delete window.__GULYAEM_DEBUG_MAP__
      instance.remove()
      map.current = null
    }
  }, [])

  useEffect(() => {
    if (!ready || !map.current) return
    ;(map.current.getSource('product-segments') as GeoJSONSource).setData(props.collection)
  }, [props.collection, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    ;(map.current.getSource('product-explored') as GeoJSONSource).setData(props.explored)
    ;(map.current.getSource('product-new') as GeoJSONSource).setData(props.newlyExplored ?? emptyLineCollection())
  }, [props.explored, props.newlyExplored, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    ;(map.current.getSource('product-route') as GeoJSONSource).setData(routeCollection(props.preview, props.routeGeometry))
    ;(map.current.getSource('product-coverage') as GeoJSONSource).setData(coverageCollection(props.preview))
    map.current.setPaintProperty('product-route-line', 'line-opacity', props.calculating ? 0.38 : 0.96)
    map.current.setPaintProperty('product-coverage-completed', 'line-opacity', props.calculating ? 0.2 : 0.9)
    map.current.setPaintProperty('product-coverage-partial', 'line-opacity', props.calculating ? 0.15 : 0.9)
  }, [props.preview, props.routeGeometry, props.calculating, ready])

  useEffect(() => {
    if (!ready || !map.current) return
    const activeIDs = new Set(props.waypoints.map((waypoint) => waypoint.id))
    for (const [id, marker] of markers.current) {
      if (!activeIDs.has(id)) {
        marker.remove()
        markers.current.delete(id)
      }
    }
    props.waypoints.forEach((waypoint, index) => {
      let marker = markers.current.get(waypoint.id)
      if (!marker) {
        const element = document.createElement('button')
        element.type = 'button'
        element.className = 'route-marker'
        element.setAttribute('aria-label', `Точка маршрута ${index + 1}`)
        marker = new Marker({ element, draggable: true })
          .setLngLat([waypoint.lon, waypoint.lat])
          .addTo(map.current!)
        const waypointID = waypoint.id
        marker.on('dragend', () => {
          const position = marker!.getLngLat()
          callbacks.current.onMoveWaypoint(waypointID, position.lat, position.lng)
        })
        markers.current.set(waypoint.id, marker)
      }
      marker.getElement().textContent = waypointLabel(index, props.waypoints.length)
      marker.getElement().setAttribute('aria-label', `Точка маршрута ${waypointLabel(index, props.waypoints.length)}`)
      marker.setLngLat([waypoint.lon, waypoint.lat])
    })
  }, [props.waypoints, ready])

  useEffect(() => {
    if (!map.current) return
    map.current.getCanvas().classList.toggle('route-map--accepting', props.acceptingPoint)
  }, [props.acceptingPoint])

  return <div ref={container} className="map product-map" aria-label="Карта для построения прогулки" />
}

function addProductLayers(instance: MapLibreMap) {
  instance.addSource('product-segments', { type: 'geojson', data: emptyCollection() })
  instance.addLayer({
    id: 'product-segments-explore', type: 'line', source: 'product-segments',
    filter: ['==', ['get', 'classification'], 'EXPLORE'],
    paint: { 'line-color': '#71827d', 'line-width': ['interpolate', ['linear'], ['zoom'], 12, 1, 17, 3.5], 'line-opacity': 0.36 },
  })
  instance.addSource('product-explored', { type: 'geojson', data: emptyLineCollection() })
  instance.addLayer({
    id: 'product-explored-line', type: 'line', source: 'product-explored',
    paint: { 'line-color': '#218a65', 'line-width': ['interpolate', ['linear'], ['zoom'], 12, 2.2, 17, 6], 'line-opacity': .82 },
  })
  instance.addLayer({
    id: 'product-segments-routable', type: 'line', source: 'product-segments',
    filter: ['==', ['get', 'classification'], 'ROUTABLE_ONLY'],
    paint: { 'line-color': '#7d8b87', 'line-width': ['interpolate', ['linear'], ['zoom'], 12, .7, 17, 2.5], 'line-opacity': 0.2 },
  })
  instance.addSource('product-coverage', { type: 'geojson', data: emptyLineCollection() })
  instance.addLayer({
    id: 'product-coverage-partial', type: 'line', source: 'product-coverage',
    filter: ['==', ['get', 'status'], 'PARTIAL'],
    paint: { 'line-color': '#f3c969', 'line-width': ['interpolate', ['linear'], ['zoom'], 12, 4, 17, 9], 'line-opacity': .9 },
  })
  instance.addLayer({
    id: 'product-coverage-completed', type: 'line', source: 'product-coverage',
    filter: ['==', ['get', 'status'], 'COMPLETED'],
    paint: { 'line-color': '#59d6a6', 'line-width': ['interpolate', ['linear'], ['zoom'], 12, 4.5, 17, 10], 'line-opacity': .9 },
  })
  instance.addSource('product-route', { type: 'geojson', data: emptyLineCollection() })
  instance.addLayer({
    id: 'product-route-casing', type: 'line', source: 'product-route',
    paint: { 'line-color': '#10201d', 'line-width': ['interpolate', ['linear'], ['zoom'], 12, 5, 17, 10], 'line-opacity': .8 },
  })
  instance.addLayer({
    id: 'product-route-line', type: 'line', source: 'product-route',
    paint: { 'line-color': '#f7fbf9', 'line-width': ['interpolate', ['linear'], ['zoom'], 12, 2.8, 17, 5.5], 'line-opacity': .96 },
  })
  instance.addSource('product-new', { type: 'geojson', data: emptyLineCollection() })
  instance.addLayer({
    id: 'product-new-line', type: 'line', source: 'product-new',
    paint: { 'line-color': '#ff8a3d', 'line-width': ['interpolate', ['linear'], ['zoom'], 12, 6, 17, 12], 'line-opacity': .96 },
  })
}

function routeCollection(preview: RoutePreview | null, geometry?: LineString): FeatureCollection<LineString> {
  const route = preview?.routing.geometry ?? geometry
  return { type: 'FeatureCollection', features: route ? [{ type: 'Feature', properties: {}, geometry: route }] : [] }
}

function coverageCollection(preview: RoutePreview | null): FeatureCollection<LineString> {
  return {
    type: 'FeatureCollection',
    features: preview?.explorationPreview.coverageSegments
      .filter((segment) => segment.status === 'COMPLETED' || segment.status === 'PARTIAL')
      .map((segment) => ({ type: 'Feature', properties: { status: segment.status }, geometry: segment.geometry })) ?? [],
  }
}

function emptyLineCollection(): FeatureCollection<LineString> {
  return { type: 'FeatureCollection', features: [] }
}
