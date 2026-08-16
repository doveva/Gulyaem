import { useCallback, useEffect, useState } from 'react'
import { API_URL } from '../geoPlayground/config'
import { CITY_ID } from '../geo'
import type { RoutePreview, WalkAggregate, WalkCompletion, Waypoint } from './types'

const ACTIVE_WALK_KEY = 'gulyaem.activeWalkId'

export type WalkFlowStatus = 'idle' | 'recovering' | 'materializing' | 'draft' | 'active' | 'review' | 'completing' | 'summary'

export function useWalkFlow(onRestore: (waypoints: Waypoint[]) => void, onExplorationChanged: () => void) {
  const [status, setStatus] = useState<WalkFlowStatus>(() => localStorage.getItem(ACTIVE_WALK_KEY) ? 'recovering' : 'idle')
  const [aggregate, setAggregate] = useState<WalkAggregate | null>(null)
  const [completion, setCompletion] = useState<WalkCompletion | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pendingRequestID, setPendingRequestID] = useState<string | null>(null)

  useEffect(() => {
    const walkID = localStorage.getItem(ACTIVE_WALK_KEY)
    if (!walkID) return
    api<WalkAggregate>(`/api/v1/walks/${walkID}`).then((value) => {
      if (value.walk.status === 'ACTIVE' || value.walk.status === 'REVIEW' || value.walk.status === 'DRAFT') {
        setAggregate(value)
        onRestore(value.route.waypoints.map((point) => ({ ...point, id: crypto.randomUUID() })))
        setStatus(value.walk.status === 'DRAFT' ? 'draft' : value.walk.status === 'ACTIVE' ? 'active' : 'review')
      } else {
        localStorage.removeItem(ACTIVE_WALK_KEY); setStatus('idle')
      }
    }).catch((cause: unknown) => { const failure = asFailure(cause); if (failure.code === 'not_found') { localStorage.removeItem(ACTIVE_WALK_KEY); setStatus('idle') } else { setError('Не удалось восстановить прогулку. Перезагрузите страницу, чтобы повторить.'); setStatus('recovering') } })
  }, [onRestore])

  const saveDraft = useCallback(async (preview: RoutePreview, waypoints: Waypoint[]) => {
    setStatus('materializing'); setError(null)
    const requestID = pendingRequestID ?? crypto.randomUUID(); setPendingRequestID(requestID)
    try {
      const created = await api<WalkAggregate>('/api/v1/walks', {
        method: 'POST', body: JSON.stringify({ clientRequestId: requestID, cityId: CITY_ID, profile: 'pedestrian',
          expectedPreviewFingerprint: preview.previewFingerprint, waypoints: waypoints.map(({ lat, lon }) => ({ lat, lon })) }),
      })
      localStorage.setItem(ACTIVE_WALK_KEY, created.walk.id); setAggregate(created)
      setPendingRequestID(null); setStatus('draft')
      return created
    } catch (cause) { const failure = asFailure(cause); setError(failure.message); setStatus('idle'); throw failure }
  }, [pendingRequestID])

  const start = useCallback(async (preview: RoutePreview, waypoints: Waypoint[]) => {
    const created = await saveDraft(preview, waypoints)
    try {
      const active = await api<WalkAggregate>(`/api/v1/walks/${created.walk.id}/start`, { method: 'POST' })
      setAggregate(active); setStatus('active')
    } catch (cause) { const failure = asFailure(cause); setError(failure.message); setStatus('draft'); throw failure }
  }, [saveDraft])

  const resume = useCallback(async () => {
    if (!aggregate) return
    setError(null)
    try { const active = await api<WalkAggregate>(`/api/v1/walks/${aggregate.walk.id}/start`, { method: 'POST' }); setAggregate(active); setPendingRequestID(null); setStatus('active') }
    catch (cause) { setError(asFailure(cause).message) }
  }, [aggregate])

  const finish = useCallback(async () => {
    if (!aggregate) return
    setError(null)
    try { const value = await api<WalkAggregate>(`/api/v1/walks/${aggregate.walk.id}/finish`, { method: 'POST' }); setAggregate(value); setStatus('review') }
    catch (cause) { setError(asFailure(cause).message) }
  }, [aggregate])

  const saveRoute = useCallback(async (preview: RoutePreview, waypoints: Waypoint[]) => {
    if (!aggregate) return
    setError(null)
    try {
      const value = await api<WalkAggregate>(`/api/v1/walks/${aggregate.walk.id}/route`, { method: 'PUT', body: JSON.stringify({ profile: 'pedestrian', expectedPreviewFingerprint: preview.previewFingerprint, waypoints: waypoints.map(({ lat, lon }) => ({ lat, lon })) }) })
      setAggregate(value)
    } catch (cause) { const failure = asFailure(cause); setError(failure.message); throw failure }
  }, [aggregate])

  const complete = useCallback(async () => {
    if (!aggregate) return
    setStatus('completing'); setError(null)
    try { const value = await api<WalkCompletion>(`/api/v1/walks/${aggregate.walk.id}/complete`, { method: 'POST' }); setCompletion(value); setStatus('summary'); localStorage.removeItem(ACTIVE_WALK_KEY); onExplorationChanged() }
    catch (cause) { setError(asFailure(cause).message); setStatus('review') }
  }, [aggregate, onExplorationChanged])

  const cancel = useCallback(async () => {
    if (!aggregate) return
    try { await api(`/api/v1/walks/${aggregate.walk.id}/cancel`, { method: 'POST' }); localStorage.removeItem(ACTIVE_WALK_KEY); setAggregate(null); setStatus('idle'); setError(null) }
    catch (cause) { setError(asFailure(cause).message) }
  }, [aggregate])

  const reset = useCallback(() => { setAggregate(null); setCompletion(null); setError(null); setStatus('idle'); onExplorationChanged() }, [onExplorationChanged])
  return { status, aggregate, completion, error, saveDraft, start, resume, finish, saveRoute, complete, cancel, reset }
}

export class APIError extends Error { constructor(public code: string, message: string) { super(message) } }
async function api<T = unknown>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, { ...init, headers: init?.body ? { 'Content-Type': 'application/json', ...init.headers } : init?.headers })
  if (!response.ok) { const body = await response.json().catch(() => ({})) as { code?: string; message?: string }; throw new APIError(body.code ?? 'request_failed', body.message ?? `API вернул ${response.status}`) }
  return response.json() as Promise<T>
}
function asFailure(value: unknown): APIError { return value instanceof APIError ? value : new APIError('network_error', value instanceof Error ? value.message : 'Сеть временно недоступна') }
