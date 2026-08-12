import { GeoPlayground } from './geoPlayground/GeoPlayground'
import { RouteBuilder } from './routeBuilder/RouteBuilder'
import { isGeoPlaygroundPath, isProductMapPath } from './routing'

export function App() {
  if (isGeoPlaygroundPath(window.location.pathname)) return <GeoPlayground />
  if (isProductMapPath(window.location.pathname)) return <RouteBuilder />
  return <main className="not-found">
    <p className="eyebrow">ГуляЕм</p>
    <h1>ГуляЕм</h1>
    <p>Соберите маршрут прогулки и посмотрите потенциальное покрытие.</p>
    <a href="/map">Открыть карту</a>
  </main>
}
