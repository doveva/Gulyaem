import { useEffect, useMemo, useState } from 'react'
import {
  CLASSIFICATIONS,
  districtQuery,
  emptyCollection,
  emptyDistrictCollection,
  segmentQuery,
  type APIErrorPayload,
  type AppliedFilters,
  type DistrictCollection,
  type SegmentCollection,
} from '../geo'
import { API_URL, type Visibility } from './config'

export interface MapViewport {
  bbox: [number, number, number, number]
  zoom: number
}

interface ViewportData {
  collection: SegmentCollection
  districtCollection: DistrictCollection
  loading: boolean
  error: string | null
}

interface ViewportResponse extends ViewportData {
  requestKey: string
}

const EMPTY_VIEWPORT_DATA: ViewportData = {
  collection: emptyCollection(),
  districtCollection: emptyDistrictCollection(),
  loading: false,
  error: null,
}

export function useViewportData(
  viewport: MapViewport | null,
  visibility: Visibility,
  filters: AppliedFilters,
  showDistricts: boolean,
): ViewportData {
  const [response, setResponse] = useState<ViewportResponse>({
    requestKey: '',
    collection: emptyCollection(),
    districtCollection: emptyDistrictCollection(),
    loading: false,
    error: null,
  })
  const request = useMemo(() => {
    const classifications = CLASSIFICATIONS.filter((value) => visibility[value])
    if (!viewport || viewport.zoom < 13 || (classifications.length === 0 && !showDistricts)) return null
    return {
      classifications,
      viewport,
      requestKey: JSON.stringify([viewport, classifications, filters, showDistricts]),
    }
  }, [viewport, visibility, filters, showDistricts])

  useEffect(() => {
    if (!request) return

    const controller = new AbortController()
    const timer = window.setTimeout(() => {
      const fetchJSON = async <T,>(url: string): Promise<T> => {
        const response = await fetch(url, { signal: controller.signal })
        if (!response.ok) {
          const payload = (await response.json().catch(() => ({}))) as APIErrorPayload
          throw new Error(payload.error?.message ?? `API вернул ${response.status}`)
        }
        return response.json() as Promise<T>
      }
      const segmentsRequest = request.classifications.length > 0
        ? fetchJSON<SegmentCollection>(`${API_URL}/api/v1/geo/segments?${segmentQuery(request.viewport.bbox, request.classifications, filters)}`)
        : Promise.resolve(emptyCollection())
      const districtsRequest = showDistricts
        ? fetchJSON<DistrictCollection>(`${API_URL}/api/v1/geo/districts?${districtQuery(request.viewport.bbox)}`)
        : Promise.resolve(emptyDistrictCollection())

      Promise.all([segmentsRequest, districtsRequest])
        .then(([collection, districtCollection]) => {
          setResponse({ requestKey: request.requestKey, collection, districtCollection, loading: false, error: null })
        })
        .catch((cause: unknown) => {
          if (controller.signal.aborted) return
          setResponse({
            requestKey: request.requestKey,
            collection: emptyCollection(),
            districtCollection: emptyDistrictCollection(),
            loading: false,
            error: cause instanceof Error ? cause.message : 'Не удалось загрузить геоданные',
          })
        })
    }, 250)

    return () => {
      window.clearTimeout(timer)
      controller.abort()
    }
  }, [request, filters, showDistricts])

  if (!request) return EMPTY_VIEWPORT_DATA
  if (response.requestKey !== request.requestKey) {
    return { ...response, loading: true, error: null }
  }
  return response
}
