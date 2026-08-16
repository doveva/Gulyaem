# Frontend

React and TypeScript UI for Gulyaem. The product `/map` route builder creates previews and owns the
Stage 3 Walk flow through completion and persistent exploration; the responsive `/debug/geo` playground keeps
the Stage 1 engineering and validation tools separate.

## Responsibility

- render the debug-first geo playground;
- adapt API and GeoJSON responses to frontend models;
- provide viewport statistics, layer and length filters, endpoints/boundary markers and a segment
  inspector.
- render district fill, boundaries, labels and a minimal district inspector.
- compare sample routes and coverage profiles and inspect coverage provenance.
- compare Valhalla, GraphHopper and OSRM geometries, waypoints and benchmark summaries.
- build a product route from start, destination and ordered intermediate waypoints;
- recalculate only after discrete edits, cancel stale requests and preserve editable waypoints on errors;
- explicitly save a DRAFT or materialize/start directly, then finish, correct and complete a Walk
  without client-authoritative geometry;
- restore DRAFT/ACTIVE/REVIEW from durable `activeWalkId` using backend state;
- render persistent explored segments, current district progress with manual refresh, and a Walk
  Summary with NEW segments and district deltas.

## Boundaries and dependencies

The frontend consumes the backend HTTP API but does not define backend domain entities. Map style
and API addresses are environment configuration. The selected local default is the public
OpenFreeMap Liberty style and can be replaced without code changes.

## Main scenarios

- open `/map`, choose **+ Прогулка**, tap start and destination, then inspect the route preview;
- start the Walk, finish into mandatory review, correct if needed, complete and inspect the reward;
- reload during ACTIVE/REVIEW and recover the same server-owned Walk;
- add, drag, remove or reorder intermediate waypoints and clear/restart the draft;
- open `/debug/geo` at the committed dense-center fixture;
- toggle `EXPLORE`, `ROUTABLE_ONLY`, `IGNORE`, districts, the base map and endpoint markers independently;
- filter by length and compare viewport count/length distributions;
- select a segment and inspect its version, reason code and typed normalization metadata;
- select a district and inspect its kind and source version; segment detail shows all intersecting districts;
- choose one of five routes, run a profile and inspect matched/unmatched and coverage segments;
- opt into source OSM metadata in non-production environments.
- enable routing comparison and toggle individual engine geometries against the reference route.

## Structure

```text
src/App.tsx                         route-level application shell
src/routeBuilder/RouteBuilder.tsx  product builder state and summary sheet
src/routeBuilder/RouteBuilderMap.tsx MapLibre route, coverage and draggable markers
src/routeBuilder/useRoutePreview.ts abortable route-preview lifecycle and stale response guard
src/routeBuilder/useWalkFlow.ts    materialization, lifecycle, completion and reload recovery
src/routeBuilder/useExploration.ts actor exploration summary and bbox overlay
src/geoPlayground/GeoPlayground.tsx feature composition and cross-panel state
src/geoPlayground/GeoMap.tsx        MapLibre lifecycle, sources, layers and map events
src/geoPlayground/useViewportData.ts debounced/abortable bbox data lifecycle
src/geoPlayground/LayerControls.tsx layer toggles, filters and viewport statistics
src/geoPlayground/SegmentInspector.tsx segment/district/coverage detail UI
src/geoPlayground/RouteAnalysisPanel.tsx sample routes and coverage analysis
src/geoPlayground/RoutingComparisonPanel.tsx Stage 1.6 engine comparison
src/geo.ts                          GeoJSON/API models and pure viewport helpers
src/routeAnalysis.ts                sample-route models and layer adapters
src/routingComparison.ts            routing report models and layer adapters
src/styles.css                      mobile-first debug UI
Dockerfile                          production static build
nginx.conf                          SPA fallback and container health endpoint
```

## Run and verify

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Open `http://localhost:5173/map` for the product flow or `/debug/geo` for engineering tools. Run
`npm run lint`, `npm test` and `npm run build` before
hand-off. With the local API and fixture running, install the pinned browser once with
`npx playwright install chromium` and run `npm run test:e2e`. Vite dependency pre-bundling excludes
`maplibre-gl` for local development; the production build emits its worker as an explicit hashed
asset and configures MapLibre with that URL.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `VITE_API_URL` | `http://localhost:8080` | browser-visible backend URL |
| `VITE_MAP_STYLE_URL` | OpenFreeMap Liberty | MapLibre style document URL |
| `VITE_CITY_ID` | seeded Saint Petersburg UUID | city selected by the engineering playground |

## Limitations and technical debt

Stage 3 still has no authentication or GPS capture; one server-configured development actor owns the
data. The browser calls only Go APIs and never Valhalla directly. Routing comparison reads a committed
static report. The debug UI overlays source/normalized routes, unmatched fragments, connectors and
completed/partial/not-covered segments and compares coverage profiles. Viewport requests are
debounced by 250 ms and stale requests are aborted. The API may ask the user to zoom in when the
viewport is larger than 25 km² or contains more than 10,000 segments; the UI does not silently
truncate data. Ordinary segment detail exposes typed normalization metadata only; raw OSM tags are
available exclusively through the non-production debug source.

## Related documents

- [`Stage 1 requirements`](../docs/stage%201/stage-1-requirements.md)
- [`Architecture contract`](../docs/stage%201/architecture-contract.md)
- [`Geo Playground bbox API ADR`](../docs/adr/0003-geo-playground-bbox-api.md)
- [`District data ADR`](../docs/adr/0004-versioned-administrative-districts.md)
- [`Route matching and coverage ADR`](../docs/adr/0005-sample-route-matching-and-radius-coverage.md)
- [`Routing engine ADR`](../docs/adr/0006-routing-engine-valhalla.md)
- [`Stage 1 validation report`](../docs/stage%201/validation-report.md)
