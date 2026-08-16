import { useState } from 'react'
import type { RouteAnalysis, SampleRoute } from '../routeAnalysis'
import { Stat } from './components'
import { formatDistance, formatNumber } from './format'

export type CoverageRequest =
  | { profile: string }
  | { profile: 'custom'; radiusMeters: number; coverageRatio: number; minRequiredMeters: number; maxRequiredMeters: number }

interface RouteAnalysisPanelProps {
  routes: SampleRoute[]
  selectedRoute: SampleRoute | null
  selectedRouteID: string
  analysis: RouteAnalysis | null
  loading: boolean
  error: string | null
  onRouteChange: (routeID: string) => void
  onAnalyze: (coverage: CoverageRequest) => void
}

export function RouteAnalysisPanel(props: RouteAnalysisPanelProps) {
  const [profile, setProfile] = useState('balanced')
  const [radius, setRadius] = useState('100')
  const [ratio, setRatio] = useState('0.4')
  const [minimum, setMinimum] = useState('15')
  const [maximum, setMaximum] = useState('80')
  const [validationError, setValidationError] = useState<string | null>(null)

  const analyze = () => {
    if (profile !== 'custom') {
      setValidationError(null)
      props.onAnalyze({ profile })
      return
    }
    const values = [radius, ratio, minimum, maximum].map(Number)
    if (values.some((value) => !Number.isFinite(value))) {
      setValidationError('Параметры custom-профиля должны быть числами')
      return
    }
    if (values[0] < 5 || values[0] > 200) {
      setValidationError('Радиус custom-профиля должен быть от 5 до 200 м')
      return
    }
    setValidationError(null)
    props.onAnalyze({
      profile: 'custom',
      radiusMeters: values[0],
      coverageRatio: values[1],
      minRequiredMeters: values[2],
      maxRequiredMeters: values[3],
    })
  }

  return <>
    <p className="panel-label">Sample route</p>
    <select value={props.selectedRouteID} onChange={(event) => props.onRouteChange(event.target.value)} aria-label="Тестовый маршрут">
      {props.routes.map((route) => <option key={route.id} value={route.id}>{route.name}</option>)}
    </select>
    {props.selectedRoute && <p className="route-description">
      {props.selectedRoute.description}
      {props.selectedRoute.intentionalUnmatched ? ' · есть намеренный unmatched-фрагмент' : ''}
    </p>}
    <div className="profile-row">
      <label>Профиль<select value={profile} onChange={(event) => setProfile(event.target.value)}>
        <option value="strict">Strict · 50 м</option>
        <option value="balanced">Balanced · 100 м</option>
        <option value="generous">Generous · 200 м</option>
        <option value="custom">Custom</option>
      </select></label>
      <button onClick={analyze} disabled={!props.selectedRoute || props.loading}>
        {props.loading ? 'Считаем…' : 'Анализировать'}
      </button>
    </div>
    {profile === 'custom' && <div className="custom-profile">
      <label>Радиус, м<input type="number" min="5" max="200" step="any" value={radius} onChange={(event) => setRadius(event.target.value)} inputMode="decimal" /></label>
      <label>Доля<input value={ratio} onChange={(event) => setRatio(event.target.value)} inputMode="decimal" /></label>
      <label>Min, м<input value={minimum} onChange={(event) => setMinimum(event.target.value)} inputMode="decimal" /></label>
      <label>Max, м<input value={maximum} onChange={(event) => setMaximum(event.target.value)} inputMode="decimal" /></label>
    </div>}
    {(validationError ?? props.error) && <p className="inline-error">{validationError ?? props.error}</p>}
    {props.analysis && <>
      <div className="coverage-legend">
        <span className="completed">Completed</span><span className="partial">Partial</span>
        <span className="not-covered">Not covered</span><span className="connector">Connector</span>
      </div>
      <div className="stats-grid route-stats">
        <Stat label="Matched" value={`${formatNumber(props.analysis.metrics.routeMatchedRatio * 100)}%`} />
        <Stat label="Completed" value={`${formatNumber(props.analysis.metrics.completedNetworkRatio * 100)}%`} />
        <Stat label="Покрыто" value={formatDistance(props.analysis.metrics.geometricCoveredLengthMeters)} />
        <Stat label="Unmatched" value={formatDistance(props.analysis.metrics.routeUnmatchedLengthMeters)} />
      </div>
    </>}
  </>
}
