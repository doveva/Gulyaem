import { useState, type ReactNode } from 'react'
import {
  CLASSIFICATIONS,
  EMPTY_STATISTICS,
  parseLength,
  type AppliedFilters,
  type SegmentStatistics,
} from '../geo'
import { CLASSIFICATION_COLORS, CLASSIFICATION_LABELS, type Visibility } from './config'
import { Stat } from './components'
import { formatDistance, formatInteger, formatNumber } from './format'

interface LayerControlsProps {
  children: ReactNode
  loading: boolean
  error: string | null
  statistics?: SegmentStatistics
  visibility: Visibility
  showBasemap: boolean
  showPoints: boolean
  showDistricts: boolean
  onVisibilityChange: (visibility: Visibility) => void
  onShowBasemapChange: (value: boolean) => void
  onShowPointsChange: (value: boolean) => void
  onShowDistrictsChange: (value: boolean) => void
  onFiltersChange: (filters: AppliedFilters) => void
}

export function LayerControls(props: LayerControlsProps) {
  const [minimumInput, setMinimumInput] = useState('')
  const [maximumInput, setMaximumInput] = useState('')
  const [filterError, setFilterError] = useState<string | null>(null)
  const statistics = props.statistics ?? EMPTY_STATISTICS

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
    props.onFiltersChange({ minLength: minimum, maxLength: maximum })
  }

  return <aside className="control-panel panel">
    <div className="panel-heading">
      <div><p className="panel-label">Видимый viewport</p><h2>Слои и фильтры</h2></div>
      <span
        className={props.loading ? 'loading-dot loading-dot--active' : 'loading-dot'}
        aria-label={props.loading ? 'Загрузка' : 'Загружено'}
      />
    </div>
    <section className="route-controls">{props.children}</section>
    <div className="classification-list">
      {CLASSIFICATIONS.map((classification) => <label className="layer-toggle" key={classification}>
        <input
          type="checkbox"
          checked={props.visibility[classification]}
          onChange={() => props.onVisibilityChange({
            ...props.visibility,
            [classification]: !props.visibility[classification],
          })}
        />
        <i style={{ backgroundColor: CLASSIFICATION_COLORS[classification] }} />
        <span>{CLASSIFICATION_LABELS[classification]}</span>
        <b>{classification === 'EXPLORE'
          ? statistics.exploreCount
          : classification === 'ROUTABLE_ONLY'
            ? statistics.routableOnlyCount
            : statistics.ignoreCount}</b>
      </label>)}
    </div>
    <div className="secondary-toggles">
      <label><input type="checkbox" checked={props.showBasemap} onChange={(event) => props.onShowBasemapChange(event.target.checked)} /> Подложка</label>
      <label><input type="checkbox" checked={props.showDistricts} onChange={(event) => props.onShowDistrictsChange(event.target.checked)} /> Районы</label>
      <label><input type="checkbox" checked={props.showPoints} onChange={(event) => props.onShowPointsChange(event.target.checked)} /> Узлы сегментов</label>
    </div>
    <div className="length-filter">
      <p className="panel-label">Длина, м</p>
      <div>
        <input inputMode="decimal" value={minimumInput} onChange={(event) => setMinimumInput(event.target.value)} placeholder="от" aria-label="Минимальная длина" />
        <span>—</span>
        <input inputMode="decimal" value={maximumInput} onChange={(event) => setMaximumInput(event.target.value)} placeholder="до" aria-label="Максимальная длина" />
        <button onClick={applyFilters}>Применить</button>
      </div>
      {filterError && <p className="inline-error">{filterError}</p>}
    </div>
    {props.error && <div className="map-error"><strong>Viewport не загружен</strong><span>{props.error}</span></div>}
    <div className="stats-grid">
      <Stat label="Сегменты" value={formatInteger(statistics.segmentsTotal)} />
      <Stat label="Всего" value={formatDistance(statistics.totalLengthMeters)} />
      <Stat label="Медиана" value={`${formatNumber(statistics.medianLengthMeters)} м`} />
      <Stat label="P95" value={`${formatNumber(statistics.p95LengthMeters)} м`} />
    </div>
    <div className="diagnostics">
      <span>&lt; 5 м <b>{statistics.shortSegmentCount}</b></span>
      <span>&gt; 500 м <b>{statistics.longSegmentCount}</b></span>
    </div>
  </aside>
}
