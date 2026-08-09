#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
output_directory=${1:-/tmp/gulyaem-spb-dense-center-refresh}
source_url='https://api.openstreetmap.org/api/0.6/map?bbox=30.3000,59.9300,30.3300,59.9450'
xml_path="${output_directory}/spb-dense-center.osm"
pbf_path="${output_directory}/spb-dense-center.osm.pbf"

mkdir -p "$output_directory"

curl -fsSL \
  -A 'Gulyaem Stage 1 fixture import (https://github.com/doveva/Gulyaem)' \
  "$source_url" \
  -o "$xml_path"

docker build -t gulyaem/osmium:bookworm "${repository_root}/infra/osmium"
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v "${output_directory}:/work" \
  gulyaem/osmium:bookworm \
  cat /work/spb-dense-center.osm \
  -c uid \
  -c user \
  -c changeset \
  -o /work/spb-dense-center.osm.pbf \
  --overwrite

echo "retrievedAt=$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
shasum -a 256 "$pbf_path"
docker run --rm \
  -v "${output_directory}:/work:ro" \
  gulyaem/osmium:bookworm \
  fileinfo -e /work/spb-dense-center.osm.pbf

echo "Candidate written outside the repository: ${pbf_path}"
echo "Review it before replacing the committed PBF and manifest."
