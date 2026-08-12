import { describe, expect, it } from 'vitest'
import { builderInstruction, builderReducer, initialBuilderState, isLatestPreviewRequest, type BuilderState } from './state'

const point = (id: string) => ({ id, lat: 59.93, lon: 30.31 })

describe('route builder state', () => {
  it('creates start and destination only after entering builder mode', () => {
    expect(builderReducer(initialBuilderState, { type: 'map-point', waypoint: point('ignored') })).toBe(initialBuilderState)
    let state = builderReducer(initialBuilderState, { type: 'start' })
    state = builderReducer(state, { type: 'map-point', waypoint: point('a') })
    expect(builderInstruction(state)).toBe('Выберите точку назначения')
    state = builderReducer(state, { type: 'map-point', waypoint: point('b') })
    expect(state.waypoints.map(({ id }) => id)).toEqual(['a', 'b'])
  })

  it('inserts an explicit intermediate point before destination', () => {
    let state: BuilderState = { mode: 'editing', waypoints: [point('a'), point('b')], addingIntermediate: false }
    state = builderReducer(state, { type: 'add-intermediate' })
    state = builderReducer(state, { type: 'map-point', waypoint: point('c') })
    expect(state.waypoints.map(({ id }) => id)).toEqual(['a', 'c', 'b'])
    expect(state.addingIntermediate).toBe(false)
  })

  it('moves, reorders and removes only intermediate points', () => {
    let state: BuilderState = { mode: 'editing', waypoints: [point('a'), point('c'), point('d'), point('b')], addingIntermediate: false }
    state = builderReducer(state, { type: 'move', id: 'c', lat: 60, lon: 31 })
    expect(state.waypoints[1]).toMatchObject({ lat: 60, lon: 31 })
    state = builderReducer(state, { type: 'reorder', id: 'd', direction: -1 })
    expect(state.waypoints.map(({ id }) => id)).toEqual(['a', 'd', 'c', 'b'])
    state = builderReducer(state, { type: 'remove', id: 'd' })
    expect(state.waypoints.map(({ id }) => id)).toEqual(['a', 'c', 'b'])
    expect(builderReducer(state, { type: 'remove', id: 'a' })).toBe(state)
  })

  it('clears the draft without leaving editing mode', () => {
    const state = builderReducer({ mode: 'editing', waypoints: [point('a'), point('b')], addingIntermediate: false }, { type: 'clear' })
    expect(state).toEqual({ mode: 'editing', waypoints: [], addingIntermediate: false })
  })
})

describe('stale response guard', () => {
  it('accepts only the active request sequence', () => {
    expect(isLatestPreviewRequest(11, 11)).toBe(true)
    expect(isLatestPreviewRequest(11, 10)).toBe(false)
  })
})
