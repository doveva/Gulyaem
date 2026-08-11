import type { DistrictProperties } from '../geo'
import type { CoverageFeature } from '../routeAnalysis'

export type PlaygroundSelection =
  | { kind: 'segment'; id: string }
  | { kind: 'district'; district: DistrictProperties }
  | { kind: 'coverage'; coverage: CoverageFeature['properties'] }
  | null
