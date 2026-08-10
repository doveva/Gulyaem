#!/usr/bin/env python3
import argparse
import json
import os
import re
import subprocess
import time
import urllib.request
from pathlib import Path


def command(*parts: str) -> str:
    return subprocess.run(parts, check=False, text=True, capture_output=True).stdout.strip()


def container_id(service: str) -> str:
    lines = command("docker", "compose", "ps", "--all", "--quiet", service).splitlines()
    return lines[0] if lines else ""


def memory_bytes(container: str) -> int:
    value = command("docker", "stats", "--no-stream", "--format", "{{.MemUsage}}", container)
    if not value:
        return 0
    match = re.match(r"\s*([0-9.]+)\s*([KMGT]?i?B)", value)
    if not match:
        return 0
    number = float(match.group(1))
    units = {"B": 1, "KB": 1000, "MB": 1000**2, "GB": 1000**3,
             "KiB": 1024, "MiB": 1024**2, "GiB": 1024**3, "TiB": 1024**4}
    return int(number * units[match.group(2)])


def directory_bytes(path: Path) -> int:
    return sum(item.stat().st_size for item in path.rglob("*") if item.is_file())


def http_ready(url: str) -> bool:
    try:
        with urllib.request.urlopen(url, timeout=1.5) as response:
            return 200 <= response.status < 300
    except Exception:
        return False


def exited(container: str) -> tuple[bool, int]:
    state = command("docker", "inspect", "--format", "{{.State.Status}} {{.State.ExitCode}}", container)
    if not state:
        return False, -1
    status, exit_code = state.split()
    return status == "exited", int(exit_code)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--service", required=True)
    parser.add_argument("--engine", required=True)
    parser.add_argument("--ready-url")
    parser.add_argument("--wait-exit", action="store_true")
    parser.add_argument("--graph-dir", required=True)
    parser.add_argument("--graph-reused", action="store_true")
    parser.add_argument("--started-ns", type=int)
    parser.add_argument("--start-service", action="store_true")
    parser.add_argument("--output", required=True)
    parser.add_argument("--timeout", type=float, default=600)
    args = parser.parse_args()

    if not args.start_service and args.started_ns is None:
        parser.error("--started-ns is required unless --start-service is used")
    started_ns = time.time_ns() if args.start_service else args.started_ns
    if args.start_service:
        subprocess.run(
            ["docker", "compose", "--profile", "routing-spike", "up", "-d", args.service],
            stdout=subprocess.DEVNULL,
            check=True,
        )

    peak = 0
    deadline = time.monotonic() + args.timeout
    current_container = ""
    while time.monotonic() < deadline:
        if not current_container:
            current_container = container_id(args.service)
        if current_container:
            peak = max(peak, memory_bytes(current_container))
            is_exited, exit_code = exited(current_container)
            if args.wait_exit:
                if is_exited:
                    if exit_code != 0:
                        raise SystemExit(f"{args.service} exited with code {exit_code}")
                    break
            elif is_exited:
                raise SystemExit(f"{args.service} exited with code {exit_code}")
            elif args.ready_url and http_ready(args.ready_url):
                break
        time.sleep(0.5)
    else:
        raise SystemExit(f"timed out waiting for {args.service}")

    peak_file = Path(args.graph_dir) / ".prepare-peak-memory-bytes"
    if args.wait_exit and peak_file.exists():
        try:
            peak = max(peak, int(peak_file.read_text().strip()))
        except ValueError:
            pass
    idle = memory_bytes(current_container) if current_container and not args.wait_exit else 0
    payload = {
        "engine": args.engine,
        "service": args.service,
        "readySeconds": (time.time_ns() - started_ns) / 1_000_000_000,
        "peakMemoryBytes": peak,
        "idleMemoryBytes": idle,
        "graphBytes": directory_bytes(Path(args.graph_dir)),
        "graphReused": args.graph_reused,
    }
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(payload, indent=2) + "\n")


if __name__ == "__main__":
    main()
