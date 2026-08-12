#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
pbf=${ROUTING_SOURCE_PBF:-"$repository_root/data/test-areas/spb-stage1-validation/spb-stage1-validation.osm.pbf"}
manifest=${ROUTING_SOURCE_MANIFEST:-"$repository_root/data/test-areas/spb-stage1-validation/manifest.json"}
valhalla_root=${VALHALLA_GRAPH_DIR:-"$repository_root/.routing/valhalla"}

checksum() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

expected=$(sed -n 's/.*"sha256"[[:space:]]*:[[:space:]]*"\([0-9a-fA-F]\{64\}\)".*/\1/p' "$manifest" | head -n 1)
if [ -z "$expected" ]; then
	echo "routing manifest does not contain a PBF sha256" >&2
	exit 1
fi
actual=$(checksum "$pbf")
if [ "$actual" != "$expected" ]; then
	echo "routing PBF checksum mismatch: got $actual, want $expected" >&2
	exit 1
fi

mkdir -p "$valhalla_root"
metadata="$valhalla_root/routing-dataset.json"
previous_source=""
previous_graph=""
if [ -f "$metadata" ]; then
	previous_source=$(sed -n 's/.*"sourceChecksum"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$metadata" | head -n 1)
	previous_graph=$(sed -n 's/.*"graphChecksum"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$metadata" | head -n 1)
fi

invalidate=false
if [ -f "$valhalla_root/valhalla_tiles.tar" ] || [ -d "$valhalla_root/valhalla_tiles" ] || [ -d "$valhalla_root/tiles" ]; then
	if [ "$previous_source" != "$actual" ] || [ -z "$previous_graph" ] || [ ! -f "$valhalla_root/valhalla_tiles.tar" ]; then
		invalidate=true
	elif [ -f "$valhalla_root/valhalla_tiles.tar" ] && [ "$(checksum "$valhalla_root/valhalla_tiles.tar")" != "$previous_graph" ]; then
		invalidate=true
	fi
fi
if [ "$invalidate" = true ]; then
	case "$valhalla_root" in
		*/.routing/valhalla|/custom_files) ;;
		*) echo "refusing to invalidate unexpected Valhalla path: $valhalla_root" >&2; exit 1 ;;
	esac
	find "$valhalla_root" -mindepth 1 -depth -delete
	echo "invalidated Valhalla graph: source or graph checksum changed"
elif [ ! -f "$valhalla_root/valhalla_tiles.tar" ] && [ ! -d "$valhalla_root/valhalla_tiles" ] && [ ! -d "$valhalla_root/tiles" ]; then
	find "$valhalla_root" -maxdepth 1 -type f -name 'routing-dataset.json' -delete
fi

valhalla_pbf="$valhalla_root/spb-stage1-validation.osm.pbf"
if [ ! -f "$valhalla_pbf" ] || [ "$(checksum "$valhalla_pbf")" != "$actual" ]; then
	cp "$pbf" "$valhalla_pbf"
fi

if [ "${ROUTING_VALHALLA_ONLY:-false}" != "true" ]; then
	mkdir -p "$repository_root/.routing/graphhopper" "$repository_root/.routing/osrm" \
		"$repository_root/.routing/setup-parts"
fi

echo "routing source ready: $pbf ($actual)"
