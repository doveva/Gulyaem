#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
pbf="$repository_root/data/test-areas/spb-stage1-validation/spb-stage1-validation.osm.pbf"
expected="05fa864bb753ffc4b2c632deae28e6b6ed80b8e677f65a9689d786596aa0e8ae"

checksum() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

actual=$(checksum "$pbf")
if [ "$actual" != "$expected" ]; then
  echo "routing PBF checksum mismatch: got $actual, want $expected" >&2
  exit 1
fi

mkdir -p "$repository_root/.routing/valhalla" "$repository_root/.routing/graphhopper" \
  "$repository_root/.routing/osrm" "$repository_root/.routing/setup-parts"

valhalla_pbf="$repository_root/.routing/valhalla/spb-stage1-validation.osm.pbf"
if [ ! -f "$valhalla_pbf" ] || [ "$(checksum "$valhalla_pbf")" != "$expected" ]; then
  cp "$pbf" "$valhalla_pbf"
fi

echo "routing fixture ready: $pbf"
