#!/usr/bin/env python3
import json
import platform
import sys
from datetime import datetime, timezone
from pathlib import Path


def main() -> None:
    root = Path(sys.argv[1])
    parts = [json.loads(path.read_text()) for path in sorted((root / "setup-parts").glob("*.json"))]
    engines: dict[str, dict] = {}
    for part in parts:
        engine = part.pop("engine")
        part.pop("service", None)
        current = engines.get(engine)
        if current is None:
            engines[engine] = part
            continue
        current["readySeconds"] += part["readySeconds"]
        current["peakMemoryBytes"] = max(current["peakMemoryBytes"], part["peakMemoryBytes"])
        current["idleMemoryBytes"] = max(current["idleMemoryBytes"], part["idleMemoryBytes"])
        current["graphBytes"] = max(current["graphBytes"], part["graphBytes"])
        current["graphReused"] = current["graphReused"] and part["graphReused"]
    payload = {
        "measuredAt": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "host": {"os": platform.system(), "architecture": platform.machine(), "release": platform.release()},
        "engines": engines,
    }
    (root / "setup-metrics.json").write_text(json.dumps(payload, indent=2) + "\n")


if __name__ == "__main__":
    main()
