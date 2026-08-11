import { GeoPlayground } from './geoPlayground/GeoPlayground'
import { isGeoPlaygroundPath } from './routing'

export function App() {
  if (isGeoPlaygroundPath(window.location.pathname)) return <GeoPlayground />
  return <main className="not-found">
    <p className="eyebrow">ГуляЕм</p>
    <h1>Geo Playground</h1>
    <p>Инженерная карта доступна по адресу /debug/geo.</p>
    <a href="/debug/geo">Открыть playground</a>
  </main>
}
