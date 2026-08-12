#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repository_root"
"$repository_root/scripts/routing/prepare.sh"

rm -f "$repository_root/.routing/setup-parts"/*.json

measure_http() {
  engine=$1
  service=$2
  url=$3
  graph_dir=$4
  marker=$5
  reused=""
  if [ -e "$marker" ]; then reused="--graph-reused"; fi
  python3 scripts/routing/measure.py --engine "$engine" --service "$service" --ready-url "$url" \
    --graph-dir "$graph_dir" $reused --start-service \
    --output ".routing/setup-parts/$service.json"
}

measure_http valhalla valhalla "http://localhost:${VALHALLA_PORT:-8002}/status" \
  "$repository_root/.routing/valhalla" "$repository_root/.routing/valhalla/valhalla_tiles.tar"

measure_http graphhopper graphhopper "http://localhost:${GRAPHHOPPER_PORT:-8989}/info" \
  "$repository_root/.routing/graphhopper" "$repository_root/.routing/graphhopper/spb-stage1-validation/properties"

osrm_reused=""
if [ -f "$repository_root/.routing/osrm/.prepared-sha256" ]; then osrm_reused="--graph-reused"; fi
python3 scripts/routing/measure.py --engine osrm --service osrm-prepare --wait-exit \
  --graph-dir "$repository_root/.routing/osrm" $osrm_reused --start-service \
  --output .routing/setup-parts/osrm-prepare.json
python3 scripts/routing/measure.py --engine osrm --service osrm \
  --ready-url "http://localhost:${OSRM_PORT:-5001}/nearest/v1/foot/30.315,59.935" \
  --graph-dir "$repository_root/.routing/osrm" $osrm_reused --start-service \
  --output .routing/setup-parts/osrm.json

python3 scripts/routing/merge_metrics.py "$repository_root/.routing"
"$repository_root/scripts/routing/finalize-metadata.sh"
echo "routing engines are ready; setup metrics: .routing/setup-metrics.json"
