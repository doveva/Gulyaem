import { useEffect, useState } from 'react'
import type { SegmentDetail } from '../geo'
import { API_URL, CLASSIFICATION_COLORS, CLASSIFICATION_LABELS } from './config'
import { Detail } from './components'
import { formatDistance, formatNumber } from './format'
import type { PlaygroundSelection } from './types'

interface SegmentInspectorProps {
  selection: PlaygroundSelection
  onClose: () => void
}

export function SegmentInspector({ selection, onClose }: SegmentInspectorProps) {
  const [response, setResponse] = useState<{
    requestKey: string
    detail: SegmentDetail | null
    error: boolean
  }>({ requestKey: '', detail: null, error: false })
  const [showDebug, setShowDebug] = useState(false)
  const segmentID = selection?.kind === 'segment' ? selection.id : null
  const requestKey = segmentID ? `${segmentID}:${showDebug}` : ''

  useEffect(() => {
    if (!segmentID) return
    const controller = new AbortController()
    fetch(`${API_URL}/api/v1/geo/segments/${segmentID}${showDebug ? '?debug=true' : ''}`, { signal: controller.signal })
      .then(async (response) => {
        if (!response.ok) throw new Error(`API вернул ${response.status}`)
        return response.json() as Promise<SegmentDetail>
      })
      .then((result) => {
        setResponse({ requestKey, detail: result, error: false })
      })
      .catch(() => {
        if (!controller.signal.aborted) setResponse({ requestKey, detail: null, error: true })
      })
    return () => controller.abort()
  }, [segmentID, showDebug, requestKey])

  const currentResponse = response.requestKey === requestKey ? response : null
  const selectedDetail = currentResponse?.detail?.id === segmentID ? currentResponse.detail : null
  const loading = Boolean(segmentID && !currentResponse)
  const district = selection?.kind === 'district' ? selection.district : null
  const coverage = selection?.kind === 'coverage' ? selection.coverage : null
  const title = coverage
    ? 'Покрытие сегмента'
    : district?.name ?? selectedDetail?.street?.name ?? (segmentID ? 'Сегмент' : 'Выберите объект')

  return <aside className={`inspector panel ${selection ? 'inspector--open' : ''}`} aria-live="polite">
    <div className="inspector-handle" />
    <div className="panel-heading">
      <div><p className="panel-label">Инспектор</p><h2>{title}</h2></div>
      {selection && <button className="icon-button" onClick={onClose} aria-label="Закрыть инспектор">×</button>}
    </div>
    {!selection && <p className="empty-message">Нажмите на цветную линию, покрытие или район, чтобы увидеть детали.</p>}
    {segmentID && loading && <p className="empty-message">Загружаем детали…</p>}
    {segmentID && currentResponse?.error && <p className="empty-message">Детали недоступны</p>}
    {district && <>
      <div className="district-summary"><span>Административный район</span></div>
      <dl className="detail-list">
        <Detail label="Тип" value={district.kind} />
        <Detail label="Источник" value={district.source} />
        <Detail label="Версия" value={district.districtDataVersionId} code />
        <Detail label="Нормализация" value={district.normalizationVersion} />
        <Detail label="External ID" value={district.externalId} code />
      </dl>
    </>}
    {coverage && <>
      <div className="segment-summary">
        <span>{coverage.status}</span>
        <strong>{formatDistance(Number(coverage.coveredMeters))}</strong>
      </div>
      <dl className="detail-list">
        <Detail label="Происхождение" value={coverage.provenance || '—'} />
        <Detail label="Требуется" value={formatDistance(Number(coverage.requiredMeters))} />
        <Detail label="Длина" value={formatDistance(Number(coverage.lengthMeters))} />
        <Detail label="ID" value={coverage.id} code />
      </dl>
    </>}
    {selectedDetail && !loading && <>
      <div className="segment-summary">
        <span style={{ color: CLASSIFICATION_COLORS[selectedDetail.classification] }}>
          {CLASSIFICATION_LABELS[selectedDetail.classification]}
        </span>
        <strong>{formatNumber(selectedDetail.lengthMeters)} м</strong>
      </div>
      <dl className="detail-list">
        <Detail label="Причина" value={selectedDetail.reasonCode} />
        <Detail label="Версия" value={`${selectedDetail.versionStatus}${selectedDetail.isCurrent ? ' · current' : ''}`} />
        <Detail label="Нормализация" value={selectedDetail.normalizationVersion} />
        <Detail label="ID" value={selectedDetail.id} code />
      </dl>
      <p className="panel-label attributes-title">Районы</p>
      {selectedDetail.districts.length === 0
        ? <p className="empty-message compact">Район не определён</p>
        : <div className="district-chips">
            {selectedDetail.districts.map((item) => <span key={item.id}>{item.name}</span>)}
          </div>}
      <p className="panel-label attributes-title">Нормализация сегмента</p>
      <dl className="attribute-list">
        <Detail label="boundaryClipped" value={selectedDetail.normalization.boundaryClipped ? 'true' : 'false'} code />
        <Detail label="warnings" value={selectedDetail.normalization.warnings.join(', ') || '—'} code />
      </dl>
      <label className="debug-toggle">
        <input type="checkbox" checked={showDebug} onChange={(event) => setShowDebug(event.target.checked)} />
        {' '}Показать OSM debug metadata
      </label>
      {showDebug && selectedDetail.debugSource && <pre className="debug-json">
        {JSON.stringify(selectedDetail.debugSource, null, 2)}
      </pre>}
    </>}
  </aside>
}
