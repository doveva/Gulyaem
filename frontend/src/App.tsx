import { useEffect, useRef, useState } from 'react'
import { Map, NavigationControl } from 'maplibre-gl'
import { isGeoPlaygroundPath } from './routing'

type ApiState = 'checking' | 'ready' | 'unavailable'

const apiURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'
const mapStyleURL =
  import.meta.env.VITE_MAP_STYLE_URL ?? 'https://tiles.openfreemap.org/styles/liberty'

export function App() {
  const mapContainer = useRef<HTMLDivElement>(null)
  const map = useRef<Map | null>(null)
  const [apiState, setApiState] = useState<ApiState>('checking')

  useEffect(() => {
    if (!mapContainer.current || map.current) return

    map.current = new Map({
      container: mapContainer.current,
      style: mapStyleURL,
      center: [30.3158, 59.9391],
      zoom: 11,
    })
    map.current.addControl(new NavigationControl(), 'bottom-right')

    return () => {
      map.current?.remove()
      map.current = null
    }
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    fetch(`${apiURL}/health/ready`, { signal: controller.signal })
      .then((response) => {
        setApiState(response.ok ? 'ready' : 'unavailable')
      })
      .catch(() => {
        if (!controller.signal.aborted) setApiState('unavailable')
      })
    return () => controller.abort()
  }, [])

  if (!isGeoPlaygroundPath(window.location.pathname)) {
    return (
      <main className="not-found">
        <p className="eyebrow">ГуляЕм</p>
        <h1>Geo Playground</h1>
        <p>Инженерная карта доступна по адресу /debug/geo.</p>
        <a href="/debug/geo">Открыть playground</a>
      </main>
    )
  }

  return (
    <main className="playground">
      <div ref={mapContainer} className="map" aria-label="Карта Санкт-Петербурга" />
      <header className="topbar">
        <div>
          <p className="eyebrow">Stage 1 · Geo Exploration</p>
          <h1>ГуляЕм</h1>
        </div>
        <div className={`api-status api-status--${apiState}`} role="status">
          <span aria-hidden="true" />
          {apiState === 'checking' && 'Проверяем API'}
          {apiState === 'ready' && 'API и PostGIS готовы'}
          {apiState === 'unavailable' && 'API недоступен'}
        </div>
      </header>
      <aside className="debug-panel">
        <p className="panel-label">Слои</p>
        <h2>Основа подключена</h2>
        <p>
          На следующем подэтапе здесь появятся версии геоданных, а затем слои собственных
          StreetSegment.
        </p>
        <dl>
          <div>
            <dt>Город</dt>
            <dd>Санкт-Петербург</dd>
          </div>
          <div>
            <dt>Карта</dt>
            <dd>MapLibre GL JS</dd>
          </div>
          <div>
            <dt>Данные</dt>
            <dd>Ожидают импорта</dd>
          </div>
        </dl>
      </aside>
    </main>
  )
}
