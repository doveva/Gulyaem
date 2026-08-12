#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
temporary_root=$(mktemp -d)
trap 'rm -rf "$temporary_root"' EXIT HUP INT TERM

source_root="$temporary_root/source"
graph_root="$temporary_root/.routing/valhalla"
mkdir -p "$source_root" "$graph_root"

checksum() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

printf 'new routing source\n' > "$source_root/source.osm.pbf"
source_checksum=$(checksum "$source_root/source.osm.pbf")
printf '{"sha256":"%s"}\n' "$source_checksum" > "$source_root/manifest.json"

printf 'stale graph\n' > "$graph_root/valhalla_tiles.tar"
stale_graph_checksum=$(checksum "$graph_root/valhalla_tiles.tar")
printf '{"sourceChecksum":"%064d","graphChecksum":"%s"}\n' 0 "$stale_graph_checksum" \
	> "$graph_root/routing-dataset.json"

ROUTING_SOURCE_PBF="$source_root/source.osm.pbf" \
ROUTING_SOURCE_MANIFEST="$source_root/manifest.json" \
VALHALLA_GRAPH_DIR="$graph_root" \
ROUTING_VALHALLA_ONLY=true \
	"$repository_root/scripts/routing/prepare.sh"

test ! -e "$graph_root/valhalla_tiles.tar"
test ! -e "$graph_root/routing-dataset.json"
test "$(checksum "$graph_root/spb-stage1-validation.osm.pbf")" = "$source_checksum"

mkdir -p "$graph_root/valhalla_tiles/0"
printf 'stale unpacked tile\n' > "$graph_root/valhalla_tiles/0/tile"
printf '{"sourceChecksum":"%s","graphChecksum":"%064d"}\n' "$source_checksum" 0 \
	> "$graph_root/routing-dataset.json"

ROUTING_SOURCE_PBF="$source_root/source.osm.pbf" \
ROUTING_SOURCE_MANIFEST="$source_root/manifest.json" \
VALHALLA_GRAPH_DIR="$graph_root" \
ROUTING_VALHALLA_ONLY=true \
	"$repository_root/scripts/routing/prepare.sh"

test ! -e "$graph_root/valhalla_tiles"
test ! -e "$graph_root/routing-dataset.json"

printf 'current graph\n' > "$graph_root/valhalla_tiles.tar"
current_graph_checksum=$(checksum "$graph_root/valhalla_tiles.tar")
printf '{"sourceChecksum":"%s","graphChecksum":"%s"}\n' "$source_checksum" "$current_graph_checksum" \
	> "$graph_root/routing-dataset.json"

ROUTING_SOURCE_PBF="$source_root/source.osm.pbf" \
ROUTING_SOURCE_MANIFEST="$source_root/manifest.json" \
VALHALLA_GRAPH_DIR="$graph_root" \
ROUTING_VALHALLA_ONLY=true \
	"$repository_root/scripts/routing/prepare.sh"

test "$(checksum "$graph_root/valhalla_tiles.tar")" = "$current_graph_checksum"
test -s "$graph_root/routing-dataset.json"

echo "routing prepare invalidation test passed"
