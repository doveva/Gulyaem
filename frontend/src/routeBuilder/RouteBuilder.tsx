import { useReducer, useState } from 'react'
import { type Visibility } from '../geoPlayground/config'
import { useViewportData, type MapViewport } from '../geoPlayground/useViewportData'
import { builderInstruction, builderReducer, initialBuilderState, waypointLabel } from './state'
import { RouteBuilderMap } from './RouteBuilderMap'
import { useRoutePreview } from './useRoutePreview'
import { formatDistance, formatDuration } from './format'

const NETWORK_VISIBILITY: Visibility = { EXPLORE: true, ROUTABLE_ONLY: true, IGNORE: false }
const NETWORK_FILTERS = { minLength: null, maxLength: null }

export function RouteBuilder() {
  const [builder, dispatch] = useReducer(builderReducer, initialBuilderState)
  const [viewport, setViewport] = useState<MapViewport | null>(null)
  const viewportData = useViewportData(viewport, NETWORK_VISIBILITY, NETWORK_FILTERS, false)
  const previewState = useRoutePreview(builder.waypoints, builder.mode === 'editing')
  const preview = previewState.preview
  const acceptingPoint = builder.mode === 'editing' && (builder.waypoints.length < 2 || builder.addingIntermediate)

  const addMapPoint = (lat: number, lon: number) => {
    if (!acceptingPoint) return
    dispatch({ type: 'map-point', waypoint: { id: crypto.randomUUID(), lat, lon } })
  }

  return <main className="route-builder">
    <RouteBuilderMap
      collection={viewportData.collection}
      waypoints={builder.waypoints}
      preview={preview}
      calculating={previewState.status === 'calculating'}
      acceptingPoint={acceptingPoint}
      onMapPoint={addMapPoint}
      onMoveWaypoint={(id, lat, lon) => dispatch({ type: 'move', id, lat, lon })}
      onViewportChange={setViewport}
    />
    <header className="product-header">
      <a href="/map" className="product-brand" aria-label="ГуляЕм — карта">Гуля<span>Ем</span></a>
      <span className="product-stage">Предпросмотр прогулки</span>
      {builder.mode === 'idle' && <button type="button" className="primary-action" onClick={() => dispatch({ type: 'start' })}>+ Прогулка</button>}
    </header>
    <section className={`route-sheet ${builder.mode === 'idle' ? 'route-sheet--idle' : ''}`} aria-label="Редактор маршрута">
      <div className="sheet-handle" aria-hidden="true" />
      <p className="route-instruction">{builderInstruction(builder)}</p>
      {builder.mode === 'idle' ? <>
        <h1>Куда пойдём сегодня?</h1>
        <p className="route-intro">Соберите пешеходный маршрут и заранее посмотрите его потенциальное исследовательское покрытие.</p>
        <button type="button" className="primary-action primary-action--wide" onClick={() => dispatch({ type: 'start' })}>+ Создать прогулку</button>
      </> : <>
        {builder.waypoints.length > 0 && <ol className="waypoint-list">
          {builder.waypoints.map((waypoint, index) => {
            const intermediate = index > 0 && index < builder.waypoints.length - 1
            return <li key={waypoint.id}>
              <span className="waypoint-badge">{waypointLabel(index, builder.waypoints.length)}</span>
              <div><strong>{index === 0 ? 'Старт' : index === builder.waypoints.length - 1 ? 'Финиш' : `Точка ${index}`}</strong><small>{waypoint.lat.toFixed(5)}, {waypoint.lon.toFixed(5)}</small></div>
              {intermediate && <div className="waypoint-actions">
                <button type="button" aria-label={`Поднять точку ${index}`} disabled={index === 1} onClick={() => dispatch({ type: 'reorder', id: waypoint.id, direction: -1 })}>↑</button>
                <button type="button" aria-label={`Опустить точку ${index}`} disabled={index === builder.waypoints.length - 2} onClick={() => dispatch({ type: 'reorder', id: waypoint.id, direction: 1 })}>↓</button>
                <button type="button" aria-label={`Удалить точку ${index}`} onClick={() => dispatch({ type: 'remove', id: waypoint.id })}>×</button>
              </div>}
            </li>
          })}
        </ol>}
        {builder.waypoints.length >= 2 && <div className="route-summary" aria-live="polite">
          {preview && <div className={previewState.status === 'calculating' ? 'preview-stale' : ''}>
            <div className="route-main-metrics"><strong>{formatDistance(preview.routing.distanceMeters)}</strong><span>≈ {formatDuration(preview.routing.durationSeconds)}</span></div>
            <p className="coverage-title">Потенциально исследуется</p>
            <div className="coverage-metrics">
              <div><strong>{preview.explorationPreview.metrics.completedSegmentCount}</strong><span>сегментов</span></div>
              <div><strong>{preview.explorationPreview.metrics.partialSegmentCount}</strong><span>частично</span></div>
              <div><strong>{Math.round(preview.explorationPreview.metrics.routeMatchedRatio * 100)}%</strong><span>сопоставлено</span></div>
            </div>
          </div>}
          {previewState.status === 'calculating' && <p className="calculation-status"><i /> Пересчитываем…</p>}
          {previewState.status === 'error' && <p className="route-error" role="alert">{previewState.error}</p>}
          {preview?.warnings.includes('low_route_match') && <p className="route-warning">Часть маршрута не удалось точно сопоставить с городской сетью.</p>}
        </div>}
        <div className="builder-actions">
          <button type="button" className={builder.addingIntermediate ? 'active' : ''} disabled={builder.waypoints.length < 2 || builder.waypoints.length >= 10} onClick={() => dispatch({ type: 'add-intermediate' })}>+ Точка</button>
          <button type="button" disabled={builder.waypoints.length === 0} onClick={() => dispatch({ type: 'clear' })}>Очистить</button>
        </div>
      </>}
      {viewportData.error && <p className="network-note">Фоновая сеть временно не загрузилась — маршрут останется доступен.</p>}
    </section>
  </main>
}
