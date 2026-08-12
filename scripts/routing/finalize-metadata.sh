#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
valhalla_root=${VALHALLA_GRAPH_DIR:-"$repository_root/.routing/valhalla"}
pbf=${ROUTING_GRAPH_PBF:-"$valhalla_root/spb-stage1-validation.osm.pbf"}
graph="$valhalla_root/valhalla_tiles.tar"
metadata="$valhalla_root/routing-dataset.json"
engine_version=${ROUTING_ENGINE_VERSION:-3.7.0}
city_id=${ROUTING_CITY_ID:-01900000-0000-7000-8000-000000000001}

checksum() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

if [ ! -s "$pbf" ]; then
	echo "Valhalla source PBF is missing: $pbf" >&2
	exit 1
fi
if [ ! -s "$graph" ]; then
	echo "Valhalla graph artifact is missing: $graph" >&2
	exit 1
fi

source_checksum=$(checksum "$pbf")
graph_checksum=$(checksum "$graph")
built_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
temporary="$valhalla_root/.routing-dataset.json.tmp.$$"
trap 'rm -f "$temporary"' EXIT HUP INT TERM
printf '%s\n' \
	'{"engine":"valhalla","engineVersion":"'"$engine_version"'","cityId":"'"$city_id"'","sourceChecksum":"'"$source_checksum"'","profile":"pedestrian","graphArtifact":"valhalla_tiles.tar","graphChecksum":"'"$graph_checksum"'","builtAt":"'"$built_at"'"}' \
	> "$temporary"
mv "$temporary" "$metadata"
trap - EXIT HUP INT TERM

echo "Valhalla graph metadata finalized: source=$source_checksum graph=$graph_checksum"
