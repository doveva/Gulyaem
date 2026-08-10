# Frontend

React and TypeScript engineering UI for inspecting Gulyaem geo data. The responsive `/debug/geo`
playground combines `StreetSegment`, districts, Stage 1.5 sample-route coverage and the static
Stage 1.6 routing-engine comparison.

## Responsibility

- render the debug-first geo playground;
- adapt API and GeoJSON responses to frontend models;
- provide viewport statistics, layer and length filters, endpoints/boundary markers and a segment
  inspector.
- render district fill, boundaries, labels and a minimal district inspector.
- compare sample routes and coverage profiles and inspect coverage provenance.
- compare Valhalla, GraphHopper and OSRM geometries, waypoints and benchmark summaries.

## Boundaries and dependencies

The frontend consumes the backend HTTP API but does not define backend domain entities. Map style
and API addresses are environment configuration. The selected local default is the public
OpenFreeMap Liberty style and can be replaced without code changes.

## Main scenarios

- open `/debug/geo` at the committed dense-center fixture;
- toggle `EXPLORE`, `ROUTABLE_ONLY`, `IGNORE`, districts, the base map and endpoint markers independently;
- filter by length and compare viewport count/length distributions;
- select a segment and inspect its version, reason code and normalized attributes;
- select a district and inspect its kind and source version; segment detail shows all intersecting districts;
- choose one of five routes, run a profile and inspect matched/unmatched and coverage segments;
- opt into source OSM metadata in non-production environments.
- enable routing comparison and toggle individual engine geometries against the reference route.

## Structure

```text
src/App.tsx       application shell and map lifecycle
src/geo.ts        GeoJSON/API models and pure viewport helpers
src/routeAnalysis.ts sample-route API models and GeoJSON layer adapters
src/routingComparison.ts routing-spike report models and engine overlay adapters
src/styles.css    mobile-first debug UI
Dockerfile        production static build
nginx.conf        SPA fallback and container health endpoint
```

## Run and verify

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Open `http://localhost:5173/debug/geo`. Run `npm run lint`, `npm test` and `npm run build` before
hand-off. Vite dependency pre-bundling excludes `maplibre-gl` for local development; the production
build emits its worker as an explicit hashed asset and configures MapLibre with that URL.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `VITE_API_URL` | `http://localhost:8080` | browser-visible backend URL |
| `VITE_MAP_STYLE_URL` | OpenFreeMap Liberty | MapLibre style document URL |
| `VITE_CITY_ID` | seeded Saint Petersburg UUID | city selected by the engineering playground |

## Limitations and technical debt

The playground has only stateless sample-route analysis, not a production Walk or cumulative
progress model. Routing comparison reads a committed static report; it never calls engines from
the browser. The UI overlays source/normalized routes, unmatched fragments, connectors and
completed/partial/not-covered segments and compares coverage profiles. Viewport requests are
debounced by 250 ms and stale requests are aborted. The API may ask the user to zoom in when the
viewport is larger than 25 km² or contains more than 10,000 segments; the UI does not silently
truncate data.

## Related documents

- [`Stage 1 requirements`](../docs/stage%201/stage-1-requirements.md)
- [`Architecture contract`](../docs/stage%201/architecture-contract.md)
- [`Geo Playground bbox API ADR`](../docs/adr/0003-geo-playground-bbox-api.md)
- [`District data ADR`](../docs/adr/0004-versioned-administrative-districts.md)
- [`Route matching and coverage ADR`](../docs/adr/0005-sample-route-matching-and-radius-coverage.md)
- [`Routing engine ADR`](../docs/adr/0006-routing-engine-valhalla.md)
