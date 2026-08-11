import { ROUTING_ENGINES, type RoutingComparison, type RoutingEngineID } from '../routingComparison'
import { ROUTING_COLORS } from './config'
import { formatDistance, formatNumber } from './format'

interface RoutingComparisonPanelProps {
  comparison: RoutingComparison | null
  selectedRouteID: string
  error: string | null
  visible: boolean
  engineVisibility: Record<RoutingEngineID, boolean>
  onVisibleChange: (value: boolean) => void
  onEngineVisibilityChange: (visibility: Record<RoutingEngineID, boolean>) => void
}

export function RoutingComparisonPanel(props: RoutingComparisonPanelProps) {
  const selectedCase = props.comparison?.cases.find((routeCase) => routeCase.routeId === props.selectedRouteID) ?? null
  return <div className="routing-comparison-controls">
    <label className="routing-master-toggle">
      <input
        type="checkbox"
        checked={props.visible}
        disabled={!props.comparison}
        onChange={(event) => props.onVisibleChange(event.target.checked)}
      /> Сравнить routing engines
    </label>
    {props.error && <p className="inline-error">{props.error}</p>}
    {props.visible && selectedCase && <>
      <p className="route-description">
        Waypoints: {selectedCase.waypoints.length} · corridor {props.comparison?.benchmark.corridorMeters} м
      </p>
      <div className="routing-engine-list">
        {ROUTING_ENGINES.map((engineID) => {
          const engine = props.comparison?.engines.find((item) => item.id === engineID)
          const result = selectedCase.results.find((item) => item.engineId === engineID)
          return <div className="routing-engine-row" key={engineID}>
            <label>
              <input
                type="checkbox"
                checked={props.engineVisibility[engineID]}
                disabled={!result?.geometry}
                onChange={() => props.onEngineVisibilityChange({
                  ...props.engineVisibility,
                  [engineID]: !props.engineVisibility[engineID],
                })}
              />
              <i style={{ backgroundColor: ROUTING_COLORS[engineID] }} />
              <strong>{engine?.name ?? engineID}</strong>
            </label>
            {result?.status === 'ok' ? <div className="routing-result-metrics">
              <span>{formatDistance(result.distanceMeters ?? 0)}</span>
              <span>p50 {formatNumber(result.latency.p50Milliseconds)} мс</span>
              <span>corridor {formatNumber((result.corridor.referenceInsideCandidateRatio ?? 0) * 100)}%</span>
              <span>segments {formatNumber((result.matcher?.routeMatchedRatio ?? 0) * 100)}%</span>
              {(result.matcher?.matchedReasonMeters.service_track ?? 0) > 0 && <span>
                service/track {formatDistance(result.matcher?.matchedReasonMeters.service_track ?? 0)}
              </span>}
            </div> : <p className="inline-error compact">{result?.error ?? 'Нет результата'}</p>}
          </div>
        })}
      </div>
    </>}
  </div>
}
