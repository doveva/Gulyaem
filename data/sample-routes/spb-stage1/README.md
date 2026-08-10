# Saint Petersburg Stage 1 sample routes

Five curated GeoJSON routes used to compare sequential map matching and radius-based exploration
coverage in the dense center, the Akademicheskaya–Grazhdansky Prospekt corridor and Sosnovka.

The lines follow the reviewed OSM snapshot while retaining small GPS-like offsets. The
`konyushennaya-capella-moyka` fixture additionally contains a deliberate ambiguous/unmatched
courtyard fragment. They are immutable input fixtures, not saved user walks.

`manifest.json` pins the expected OSM source checksum and normalization rules. Analysis remains
available against a different current version, but the API reports a fixture-version warning.

## Attribution

The routes are derived from © OpenStreetMap contributors data under ODbL 1.0.
