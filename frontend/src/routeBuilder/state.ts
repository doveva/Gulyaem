import type { Waypoint } from './types'

export type BuilderMode = 'idle' | 'editing'

export interface BuilderState {
  mode: BuilderMode
  waypoints: Waypoint[]
  addingIntermediate: boolean
}

export const initialBuilderState: BuilderState = {
  mode: 'idle',
  waypoints: [],
  addingIntermediate: false,
}

export type BuilderAction =
  | { type: 'start' }
  | { type: 'map-point'; waypoint: Waypoint }
  | { type: 'add-intermediate' }
  | { type: 'move'; id: string; lat: number; lon: number }
  | { type: 'remove'; id: string }
  | { type: 'reorder'; id: string; direction: -1 | 1 }
  | { type: 'clear' }
  | { type: 'restore'; waypoints: Waypoint[] }
  | { type: 'reset' }

export function builderReducer(state: BuilderState, action: BuilderAction): BuilderState {
  switch (action.type) {
    case 'start':
      return { mode: 'editing', waypoints: [], addingIntermediate: false }
    case 'map-point': {
      if (state.mode !== 'editing' || state.waypoints.length >= 10) return state
      if (state.waypoints.length < 2) {
        return { ...state, waypoints: [...state.waypoints, action.waypoint] }
      }
      if (!state.addingIntermediate) return state
      return {
        ...state,
        addingIntermediate: false,
        waypoints: [...state.waypoints.slice(0, -1), action.waypoint, state.waypoints.at(-1)!],
      }
    }
    case 'add-intermediate':
      return state.waypoints.length >= 2 && state.waypoints.length < 10
        ? { ...state, addingIntermediate: true }
        : state
    case 'move':
      return {
        ...state,
        addingIntermediate: false,
        waypoints: state.waypoints.map((waypoint) => waypoint.id === action.id
          ? { ...waypoint, lat: action.lat, lon: action.lon }
          : waypoint),
      }
    case 'remove': {
      const index = state.waypoints.findIndex((waypoint) => waypoint.id === action.id)
      if (index <= 0 || index >= state.waypoints.length - 1) return state
      return { ...state, addingIntermediate: false, waypoints: state.waypoints.filter((waypoint) => waypoint.id !== action.id) }
    }
    case 'reorder': {
      const index = state.waypoints.findIndex((waypoint) => waypoint.id === action.id)
      const target = index + action.direction
      if (index <= 0 || index >= state.waypoints.length - 1 || target <= 0 || target >= state.waypoints.length - 1) return state
      const waypoints = [...state.waypoints]
      ;[waypoints[index], waypoints[target]] = [waypoints[target], waypoints[index]]
      return { ...state, addingIntermediate: false, waypoints }
    }
    case 'clear':
      return { mode: 'editing', waypoints: [], addingIntermediate: false }
    case 'restore':
      return { mode: 'editing', waypoints: action.waypoints, addingIntermediate: false }
    case 'reset':
      return initialBuilderState
  }
}

export function builderInstruction(state: BuilderState): string {
  if (state.mode === 'idle') return 'Создайте прогулку и отметьте маршрут на карте'
  if (state.waypoints.length === 0) return 'Выберите начало маршрута'
  if (state.waypoints.length === 1) return 'Выберите точку назначения'
  if (state.addingIntermediate) return 'Выберите дополнительную точку на карте'
  return 'Маршрут можно менять: перетащите точку или добавьте новую'
}

export function waypointLabel(index: number, count: number): string {
  if (index === 0) return 'A'
  if (index === count - 1) return 'B'
  return String(index)
}

export function isLatestPreviewRequest(activeSequence: number, responseSequence: number): boolean {
  return activeSequence === responseSequence
}
