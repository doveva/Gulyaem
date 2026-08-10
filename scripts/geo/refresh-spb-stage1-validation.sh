#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
output_directory=${1:-/tmp/gulyaem-spb-stage1-validation-refresh}
image='gulyaem/osmium:bookworm'
maximum_bytes=$((20 * 1024 * 1024))

mkdir -p "$output_directory"

fetch() {
  name=$1
  bbox=$2
  curl -fsSL \
    -A 'Gulyaem Stage 1 fixture import (https://github.com/doveva/Gulayem)' \
    "https://api.openstreetmap.org/api/0.6/map?bbox=${bbox}" \
    -o "${output_directory}/${name}.osm"
}

fetch dense-center '30.3000,59.9300,30.3300,59.9450'
fetch kalininsky-south '30.3850,60.0060,30.4100,60.0240'
fetch kalininsky-north '30.4000,60.0220,30.4250,60.0410'
fetch sosnovka-west '30.3260,60.0080,30.3500,60.0360'
fetch sosnovka-east '30.3480,60.0080,30.3730,60.0360'

docker build -t "$image" "${repository_root}/infra/osmium"
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v "${output_directory}:/work" \
  "$image" \
  merge \
  /work/dense-center.osm \
  /work/kalininsky-south.osm \
  /work/kalininsky-north.osm \
  /work/sosnovka-west.osm \
  /work/sosnovka-east.osm \
  -o /work/spb-stage1-validation.osm.pbf \
  --overwrite

pbf_path="${output_directory}/spb-stage1-validation.osm.pbf"
pbf_bytes=$(wc -c < "$pbf_path" | tr -d ' ')
if [ "$pbf_bytes" -gt "$maximum_bytes" ]; then
  echo "Candidate is ${pbf_bytes} bytes and exceeds the 20 MB review gate." >&2
  exit 1
fi

echo "retrievedAt=$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
echo "sizeBytes=${pbf_bytes}"
shasum -a 256 "$pbf_path"
docker run --rm \
  -v "${output_directory}:/work:ro" \
  "$image" \
  check-refs /work/spb-stage1-validation.osm.pbf
docker run --rm \
  -v "${output_directory}:/work:ro" \
  "$image" \
  fileinfo -e /work/spb-stage1-validation.osm.pbf

echo "Candidate written outside the repository: ${pbf_path}"
echo "Review it before replacing the committed PBF and manifest."
