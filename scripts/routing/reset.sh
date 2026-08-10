#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
target="$repository_root/.routing"
case "$target" in
  "$repository_root/.routing") ;;
  *) echo "refusing to reset unexpected path: $target" >&2; exit 1 ;;
esac

docker compose --project-directory "$repository_root" --profile routing-spike stop valhalla graphhopper osrm >/dev/null 2>&1 || true
if [ -d "$target" ]; then
  find "$target" -mindepth 1 -depth -delete
fi
echo "removed generated routing graphs and local measurements from $target"
