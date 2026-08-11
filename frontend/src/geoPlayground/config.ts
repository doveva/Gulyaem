import type { Classification } from '../geo'
import type { RoutingEngineID } from '../routingComparison'

export type Visibility = Record<Classification, boolean>

export const API_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'
export const MAP_STYLE_URL =
  import.meta.env.VITE_MAP_STYLE_URL ?? 'https://tiles.openfreemap.org/styles/liberty'

export const LINE_LAYERS: Record<Classification, string> = {
  EXPLORE: 'segments-explore',
  ROUTABLE_ONLY: 'segments-routable',
  IGNORE: 'segments-ignore',
}

export const CLASSIFICATION_LABELS: Record<Classification, string> = {
  EXPLORE: 'Исследуемые',
  ROUTABLE_ONLY: 'Только связность',
  IGNORE: 'Исключённые',
}

export const CLASSIFICATION_COLORS: Record<Classification, string> = {
  EXPLORE: '#35d3b4',
  ROUTABLE_ONLY: '#f0b34d',
  IGNORE: '#b77774',
}

export const COVERAGE_LAYER_IDS = [
  'coverage-not-covered', 'coverage-partial', 'coverage-completed', 'coverage-connector',
]

export const ROUTING_LAYER_IDS: Record<RoutingEngineID, string> = {
  valhalla: 'routing-valhalla',
  graphhopper: 'routing-graphhopper',
  osrm: 'routing-osrm',
}

export const ROUTING_COLORS: Record<RoutingEngineID, string> = {
  valhalla: '#ff9e64',
  graphhopper: '#58b9ff',
  osrm: '#ff6fae',
}
