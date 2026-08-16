import { useCallback, useEffect, useMemo, useReducer, useState, type Dispatch } from 'react'
import { type Visibility } from '../geoPlayground/config'
import { useViewportData, type MapViewport } from '../geoPlayground/useViewportData'
import { builderInstruction, builderReducer, initialBuilderState, waypointLabel } from './state'
import { RouteBuilderMap } from './RouteBuilderMap'
import { useRoutePreview } from './useRoutePreview'
import { formatDistance, formatDuration } from './format'
import { APIError, useWalkFlow } from './useWalkFlow'
import { useExploration } from './useExploration'

const NETWORK_VISIBILITY: Visibility = { EXPLORE: true, ROUTABLE_ONLY: true, IGNORE: false }
const NETWORK_FILTERS = { minLength: null, maxLength: null }

export function RouteBuilder() {
  const [builder, dispatch] = useReducer(builderReducer, initialBuilderState)
  const [viewport, setViewport] = useState<MapViewport | null>(null)
  const [previewRefresh, setPreviewRefresh] = useState(0)
  const [explorationRefresh, setExplorationRefresh] = useState(0)
  const restoreWaypoints = useCallback((waypoints: import('./types').Waypoint[]) => dispatch({ type: 'restore', waypoints }), [])
  const explorationChanged = useCallback(() => setExplorationRefresh(value => value + 1), [])
  const walk = useWalkFlow(restoreWaypoints, explorationChanged)
  const viewportData = useViewportData(viewport, NETWORK_VISIBILITY, NETWORK_FILTERS, false)
  const exploration = useExploration(viewport, explorationRefresh)
  const canEdit = walk.status === 'idle' || walk.status === 'review'
  const previewState = useRoutePreview(builder.waypoints, builder.mode === 'editing' && canEdit, previewRefresh)
  const preview = previewState.preview
  const acceptingPoint = canEdit && builder.mode === 'editing' && (builder.waypoints.length < 2 || builder.addingIntermediate)
  const correctionDirty = useMemo(() => {
    if (walk.status !== 'review' || !walk.aggregate) return false
    return JSON.stringify(builder.waypoints.map(({ lat, lon }) => ({ lat, lon }))) !== JSON.stringify(walk.aggregate.route.waypoints)
  }, [builder.waypoints, walk.aggregate, walk.status])

  const addMapPoint = (lat: number, lon: number) => {
    if (!acceptingPoint) return
    dispatch({ type: 'map-point', waypoint: { id: crypto.randomUUID(), lat, lon } })
  }
  const startWalk = async () => {
    if (!preview || previewState.status !== 'ready') return
    try { await walk.start(preview, builder.waypoints) } catch (cause) { if (cause instanceof APIError && cause.code === 'route_preview_stale') setPreviewRefresh(value => value + 1) }
  }
  const saveDraft = async () => {
    if (!preview || previewState.status !== 'ready') return
    try { await walk.saveDraft(preview, builder.waypoints) } catch (cause) { if (cause instanceof APIError && cause.code === 'route_preview_stale') setPreviewRefresh(value => value + 1) }
  }
  const saveCorrection = async () => {
    if (!preview || previewState.status !== 'ready') return
    try { await walk.saveRoute(preview, builder.waypoints) } catch (cause) { if (cause instanceof APIError && cause.code === 'route_preview_stale') setPreviewRefresh(value => value + 1) }
  }
  const returnToMap = () => { walk.reset(); dispatch({ type: 'reset' }) }
  const mapPreview = canEdit ? preview : null
  const routeGeometry = canEdit ? undefined : walk.aggregate?.route.geometry
  const stageLabel = walk.status === 'draft' ? 'Прогулка готова' : walk.status === 'active' ? 'Активная прогулка' : walk.status === 'review' ? 'Проверка маршрута' : walk.status === 'summary' ? 'Итоги прогулки' : 'Карта исследования'

  return <main className="route-builder">
    <RouteBuilderMap collection={viewportData.collection} waypoints={canEdit ? builder.waypoints : []}
      preview={mapPreview} routeGeometry={routeGeometry} explored={exploration.segments} newlyExplored={walk.completion?.exploration.newSegments}
      calculating={previewState.status === 'calculating'} acceptingPoint={acceptingPoint} onMapPoint={addMapPoint}
      onMoveWaypoint={(id, lat, lon) => canEdit && dispatch({ type: 'move', id, lat, lon })} onViewportChange={setViewport} />
    <header className="product-header"><a href="/map" className="product-brand" aria-label="ГуляЕм — карта">Гуля<span>Ем</span></a><span className="product-stage">{stageLabel}</span></header>
    <section className={`route-sheet ${walk.status === 'idle' && builder.mode === 'idle' ? 'route-sheet--idle' : ''}`} aria-label="Прогулка">
      <div className="sheet-handle" aria-hidden="true" />
      {(walk.status === 'recovering') && <><h1>Возвращаем прогулку…</h1><p className="route-intro">{walk.error ?? 'Сверяем состояние с сервером.'}</p></>}
      {walk.status === 'idle' && builder.mode === 'idle' && <IdlePanel exploration={exploration} onStart={() => dispatch({ type: 'start' })} onRefresh={explorationChanged} />}
      {walk.status === 'idle' && builder.mode === 'editing' && <><EditorPanel builder={builder} dispatch={dispatch} previewState={previewState} onPrimary={startWalk} primaryLabel="Начать прогулку" busy={false} error={walk.error} /><button type="button" className="text-action" disabled={previewState.status !== 'ready'} onClick={saveDraft}>Сохранить черновик</button></>}
      {walk.status === 'materializing' && <><h1>Готовим прогулку</h1><p className="route-intro">Повторно проверяем маршрут и сохраняем доверенную геометрию.</p><button className="primary-action primary-action--wide" disabled>Подождите…</button></>}
      {walk.status === 'draft' && walk.aggregate && <><p className="route-instruction">Маршрут уже сохранён, запуск можно безопасно повторить</p><h1>Прогулка готова</h1><p className="route-intro">Предыдущая попытка запуска не завершилась. Повтор не создаст дубликат.</p>{walk.error && <p className="route-error">{walk.error}</p>}<button className="primary-action primary-action--wide" onClick={walk.resume}>Начать сохранённую прогулку</button><button className="text-action" onClick={walk.cancel}>Отменить прогулку</button></>}
      {walk.status === 'active' && walk.aggregate && <ActivePanel aggregate={walk.aggregate} onFinish={walk.finish} onCancel={walk.cancel} error={walk.error} />}
      {walk.status === 'review' && walk.aggregate && <><p className="route-instruction">Проверьте финальный маршрут перед начислением прогресса</p><h1>Маршрут прогулки</h1><p className="route-intro">Можно перетащить точки, сохранить исправление и только затем подтвердить исследование.</p><EditorPanel builder={builder} dispatch={dispatch} previewState={previewState} onPrimary={saveCorrection} primaryLabel="Сохранить исправление" busy={false} compact error={walk.error} /><button type="button" className="primary-action primary-action--wide" disabled={correctionDirty || previewState.status === 'calculating'} onClick={walk.complete}>{correctionDirty ? 'Сначала сохраните маршрут' : 'Подтвердить исследование'}</button><button type="button" className="text-action" onClick={walk.cancel}>Отменить прогулку</button></>}
      {walk.status === 'completing' && <><h1>Обновляем карту</h1><p className="route-intro">Фиксируем новые и повторно исследованные улицы одной транзакцией.</p></>}
      {walk.status === 'summary' && walk.completion && <SummaryPanel completion={walk.completion} onClose={returnToMap} />}
      {viewportData.error && <p className="network-note">Фоновая сеть временно не загрузилась — прогулка останется доступна.</p>}
    </section>
  </main>
}

function IdlePanel({ exploration, onStart, onRefresh }: { exploration: ReturnType<typeof useExploration>; onStart: () => void; onRefresh: () => void }) {
  return <><p className="route-instruction">Исследуйте город прогулка за прогулкой</p><h1>Куда пойдём сегодня?</h1>
    {exploration.summary && <div className="progress-card"><strong>{Math.round(exploration.summary.city.percentage * 1000) / 10}%</strong><span>{formatDistance(exploration.summary.city.exploredLengthMeters)} исследовано · {exploration.summary.city.exploredSegmentsCount} сегментов</span></div>}
    {exploration.summary?.districts.map(district => <div className="district-delta" key={district.districtId}><strong>{district.name}</strong><span>{(district.percentage * 100).toFixed(1)}% · {formatDistance(district.exploredLengthMeters)}</span></div>)}
    {exploration.error && <p className="route-error" role="alert">{exploration.error}</p>}
    <button type="button" className="text-action" onClick={onRefresh}>Обновить прогресс</button>
    <p className="route-intro">Соберите маршрут, пройдите его и откройте исследованные улицы на своей карте.</p><button type="button" className="primary-action primary-action--wide" onClick={onStart}>+ Создать прогулку</button></>
}

function EditorPanel({ builder, dispatch, previewState, onPrimary, primaryLabel, busy, compact = false, error }: { builder: typeof initialBuilderState; dispatch: Dispatch<Parameters<typeof builderReducer>[1]>; previewState: ReturnType<typeof useRoutePreview>; onPrimary: () => void; primaryLabel: string; busy: boolean; compact?: boolean; error: string | null }) {
  const preview = previewState.preview
  return <div className={compact ? 'editor-compact' : ''}><p className="route-instruction">{builderInstruction(builder)}</p>{builder.waypoints.length > 0 && <ol className="waypoint-list">{builder.waypoints.map((waypoint, index) => { const intermediate = index > 0 && index < builder.waypoints.length - 1; return <li key={waypoint.id}><span className="waypoint-badge">{waypointLabel(index, builder.waypoints.length)}</span><div><strong>{index === 0 ? 'Старт' : index === builder.waypoints.length - 1 ? 'Финиш' : `Точка ${index}`}</strong><small>{waypoint.lat.toFixed(5)}, {waypoint.lon.toFixed(5)}</small></div>{intermediate && <div className="waypoint-actions"><button type="button" aria-label={`Поднять точку ${index}`} disabled={index === 1} onClick={() => dispatch({ type: 'reorder', id: waypoint.id, direction: -1 })}>↑</button><button type="button" aria-label={`Опустить точку ${index}`} disabled={index === builder.waypoints.length - 2} onClick={() => dispatch({ type: 'reorder', id: waypoint.id, direction: 1 })}>↓</button><button type="button" aria-label={`Удалить точку ${index}`} onClick={() => dispatch({ type: 'remove', id: waypoint.id })}>×</button></div>}</li>})}</ol>}
    {builder.waypoints.length >= 2 && <div className="route-summary">{preview && <div className={previewState.status === 'calculating' ? 'preview-stale' : ''}><div className="route-main-metrics"><strong>{formatDistance(preview.routing.distanceMeters)}</strong><span>≈ {formatDuration(preview.routing.durationSeconds)}</span></div><p className="coverage-title">Потенциально исследуется</p><div className="coverage-metrics"><div><strong>{preview.explorationPreview.metrics.completedSegmentCount}</strong><span>сегментов</span></div><div><strong>{preview.explorationPreview.metrics.partialSegmentCount}</strong><span>частично</span></div><div><strong>{Math.round(preview.explorationPreview.metrics.routeMatchedRatio * 100)}%</strong><span>сопоставлено</span></div></div></div>}{previewState.status === 'calculating' && <p className="calculation-status"><i /> Пересчитываем…</p>}{previewState.status === 'error' && <p className="route-error" role="alert">{previewState.error}</p>}</div>}
    <div className="builder-actions"><button type="button" className={builder.addingIntermediate ? 'active' : ''} disabled={builder.waypoints.length < 2 || builder.waypoints.length >= 10} onClick={() => dispatch({ type: 'add-intermediate' })}>+ Точка</button><button type="button" disabled={builder.waypoints.length === 0} onClick={() => dispatch({ type: 'clear' })}>Очистить</button></div>
    {error && <p className="route-error" role="alert">{error}</p>}<button type="button" className="primary-action primary-action--wide" disabled={busy || previewState.status !== 'ready'} onClick={onPrimary}>{primaryLabel}</button></div>
}

function ActivePanel({ aggregate, onFinish, onCancel, error }: { aggregate: import('./types').WalkAggregate; onFinish: () => void; onCancel: () => void; error: string | null }) {
  const [now, setNow] = useState(() => new Date(aggregate.walk.startedAt!).getTime()); useEffect(() => { const timer = window.setInterval(() => setNow(Date.now()), 1000); return () => clearInterval(timer) }, [])
  const elapsed = Math.max(0, (now - new Date(aggregate.walk.startedAt!).getTime()) / 1000)
  return <><p className="route-instruction">Прогулка началась по времени сервера</p><h1>Идём по маршруту</h1><div className="active-metrics"><div><strong>{formatDuration(elapsed)}</strong><span>прошло</span></div><div><strong>{formatDistance(aggregate.route.distanceMeters)}</strong><span>длина маршрута</span></div></div><p className="route-intro">GPS пока не записывается. Завершите прогулку, когда будете готовы проверить маршрут.</p>{error && <p className="route-error">{error}</p>}<button className="primary-action primary-action--wide" onClick={onFinish}>Завершить и проверить</button><button className="text-action" onClick={onCancel}>Отменить прогулку</button></>
}

function SummaryPanel({ completion, onClose }: { completion: import('./types').WalkCompletion; onClose: () => void }) {
  const e = completion.exploration
  return <><p className="route-instruction">Карта исследования обновлена</p><h1>{e.newSegmentsCount === 0 ? 'Знакомый маршрут' : 'Новые улицы открыты!'}</h1><div className="active-metrics"><div><strong>{e.newSegmentsCount}</strong><span>новых сегментов</span></div><div><strong>{formatDistance(e.newNetworkLengthMeters)}</strong><span>новой сети</span></div></div><p className="route-intro">Длина новой сети — это исследованные городские сегменты, а не GPS-дистанция прогулки.</p>{e.districts.map(d => <div className="district-delta" key={d.districtId}><strong>{d.name}</strong><span>{(d.percentageBefore * 100).toFixed(1)}% → {(d.percentageAfter * 100).toFixed(1)}%</span></div>)}<button className="primary-action primary-action--wide" onClick={onClose}>Вернуться к карте</button></>
}
