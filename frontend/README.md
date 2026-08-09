# Frontend

React and TypeScript engineering UI for inspecting Gulyaem geo data. Stage 1.1 establishes the
mobile-first `/debug/geo` shell, renders Saint Petersburg with MapLibre GL JS and reports backend
readiness.

## Responsibility

- render the debug-first geo playground;
- adapt API and GeoJSON responses to frontend models;
- provide map layers, filters and inspectors added during later Stage 1 increments.

## Boundaries and dependencies

The frontend consumes the backend HTTP API but does not define backend domain entities. Map style
and API addresses are environment configuration. The selected local default is the public
OpenFreeMap Liberty style and can be replaced without code changes.

## Main scenarios

- open `/debug/geo` and interact with the base map;
- see whether the API and PostGIS readiness endpoint is available;
- use the placeholder debug panel that later stages extend with geo layers.

## Structure

```text
src/App.tsx       application shell and map lifecycle
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
hand-off.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `VITE_API_URL` | `http://localhost:8080` | browser-visible backend URL |
| `VITE_MAP_STYLE_URL` | OpenFreeMap Liberty | MapLibre style document URL |

## Limitations and technical debt

Stage 1.1 contains only the base map and readiness state. StreetSegment layers, viewport data,
filters and inspectors belong to subsequent Stage 1 increments.

## Related documents

- [`Stage 1 requirements`](../docs/stage%201/stage-1-requirements.md)
- [`Architecture contract`](../docs/stage%201/architecture-contract.md)
