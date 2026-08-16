import { useEffect, useRef, useState } from 'react'
import { CITY_ID } from '../geo'
import { API_URL } from '../geoPlayground/config'
import type { RoutePreview, RoutePreviewErrorPayload, Waypoint } from './types'
import { isLatestPreviewRequest } from './state'

export type PreviewStatus = 'empty' | 'calculating' | 'ready' | 'error'

export interface PreviewState {
  status: PreviewStatus
  preview: RoutePreview | null
  error: string | null
}

export function useRoutePreview(waypoints: Waypoint[], enabled: boolean, refreshToken = 0): PreviewState {
  const sequence = useRef(0)
  const [response, setResponse] = useState<{ requestKey: string; preview: RoutePreview | null; error: string | null }>({
    requestKey: '', preview: null, error: null,
  })
  const signature = waypoints.map(({ id, lat, lon }) => `${id}:${lat}:${lon}`).join('|')
  const requestKey = enabled && waypoints.length >= 2 ? `${signature}@${refreshToken}` : ''

  useEffect(() => {
    if (!requestKey) {
      sequence.current++
      return
    }
    const requestSequence = ++sequence.current
    const controller = new AbortController()
    fetch(`${API_URL}/api/v1/route-previews`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      signal: controller.signal,
      body: JSON.stringify({
        cityId: CITY_ID,
        profile: 'pedestrian',
        waypoints: waypoints.map(({ lat, lon }) => ({ lat, lon })),
      }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const payload = (await response.json().catch(() => ({}))) as RoutePreviewErrorPayload
          const error = new Error(payload.message ?? payload.error?.message ?? `API вернул ${response.status}`)
          error.name = payload.code ?? payload.error?.code ?? 'route_preview_failed'
          throw error
        }
        return response.json() as Promise<RoutePreview>
      })
      .then((preview) => {
        if (isLatestPreviewRequest(sequence.current, requestSequence)) setResponse({ requestKey, preview, error: null })
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted || !isLatestPreviewRequest(sequence.current, requestSequence)) return
        const message = cause instanceof Error ? errorMessage(cause.name, cause.message) : 'Не удалось построить маршрут.'
        setResponse({ requestKey, preview: null, error: message })
      })
    return () => controller.abort()
  // signature captures completed waypoint edits without routing on pointer movement.
  }, [requestKey, waypoints])

  if (!requestKey) return { status: 'empty', preview: null, error: null }
  if (response.requestKey !== requestKey) return { status: 'calculating', preview: response.preview, error: null }
  if (response.error) return { status: 'error', preview: null, error: response.error }
  return { status: 'ready', preview: response.preview, error: null }
}

function errorMessage(code: string, fallback: string): string {
  switch (code) {
    case 'route_not_found': return 'Не удалось построить пешеходный маршрут между выбранными точками.'
    case 'routing_unavailable':
    case 'routing_timeout': return 'Построение маршрута временно недоступно. Попробуйте изменить точки.'
    case 'routing_geo_version_mismatch': return 'Routing graph не соответствует текущим геоданным.'
    default: return fallback
  }
}
