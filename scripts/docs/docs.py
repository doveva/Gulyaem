#!/usr/bin/env python3
"""Local docs-as-code index and code-to-README checks."""

from __future__ import annotations

import argparse
import fnmatch
import json
import os
import subprocess
import sys
from pathlib import Path, PurePosixPath
from urllib.parse import quote


INDEX_START = "<!-- docs:index:start -->"
INDEX_END = "<!-- docs:index:end -->"


def git(*args: str, check: bool = True) -> str:
    result = subprocess.run(
        ["git", *args],
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if check and result.returncode != 0:
        raise RuntimeError(result.stderr.strip() or "Git command failed")
    return result.stdout


def repository_root() -> Path:
    return Path(git("rev-parse", "--show-toplevel").strip()).resolve()


def markdown_title(path: Path) -> str:
    try:
        with path.open(encoding="utf-8") as document:
            for line in document:
                if line.startswith("# "):
                    return line[2:].strip()
    except UnicodeDecodeError:
        pass
    return path.stem.replace("-", " ").replace("_", " ").title()


def documentation_files(root: Path) -> list[Path]:
    files: set[Path] = set()
    for path in (root / "docs").rglob("*.md"):
        if path != root / "docs" / "README.md":
            files.add(path)
    for path in root.rglob("README.md"):
        if path in {root / "README.md", root / "docs" / "README.md"}:
            continue
        if any(part in {".git", "node_modules", "vendor", "dist", "build"} for part in path.parts):
            continue
        files.add(path)
    return sorted(files, key=lambda item: item.relative_to(root).as_posix().lower())


def render_index(root: Path, target: Path, files: list[Path]) -> str:
    grouped: dict[str, list[Path]] = {}
    for path in files:
        relative = path.relative_to(root)
        group = relative.parent.as_posix()
        grouped.setdefault(group, []).append(path)

    lines = ["_Этот блок сформирован автоматически. Не редактируйте его вручную._", ""]
    for group, paths in grouped.items():
        lines.append(f"### `{group}/`")
        lines.append("")
        for path in paths:
            link = os.path.relpath(path, target.parent).replace(os.sep, "/")
            lines.append(f"- [{markdown_title(path)}]({quote(link, safe='/-._~')})")
        lines.append("")
    return "\n".join(lines).rstrip()


def replace_index(document: str, rendered: str, path: Path) -> str:
    if document.count(INDEX_START) != 1 or document.count(INDEX_END) != 1:
        raise RuntimeError(f"{path}: expected exactly one documentation index block")
    before, remainder = document.split(INDEX_START, 1)
    _, after = remainder.split(INDEX_END, 1)
    return f"{before}{INDEX_START}\n{rendered}\n{INDEX_END}{after}"


def update_indexes(root: Path, check_only: bool) -> int:
    files = documentation_files(root)
    stale: list[Path] = []
    for target in (root / "README.md", root / "docs" / "README.md"):
        current = target.read_text(encoding="utf-8")
        expected = replace_index(current, render_index(root, target, files), target)
        if current == expected:
            continue
        stale.append(target)
        if not check_only:
            target.write_text(expected, encoding="utf-8")

    if stale and check_only:
        print("Documentation index is stale:")
        for path in stale:
            print(f"  - {path.relative_to(root)}")
        print("Run: python3 scripts/docs/docs.py index")
        return 1
    if stale:
        print("Updated documentation index:")
        for path in stale:
            print(f"  - {path.relative_to(root)}")
    else:
        print("Documentation index is up to date.")
    return 0


def load_config(root: Path) -> dict[str, list[str]]:
    path = root / "docs" / "docs-config.json"
    return json.loads(path.read_text(encoding="utf-8"))


def changed_files(root: Path, base: str | None) -> set[str]:
    commands = [
        ("diff", "--name-only", "--diff-filter=ACMR"),
        ("diff", "--cached", "--name-only", "--diff-filter=ACMR"),
        ("ls-files", "--others", "--exclude-standard"),
    ]
    if base:
        commands.append(("diff", "--name-only", "--diff-filter=ACMR", f"{base}...HEAD"))

    paths: set[str] = set()
    for command in commands:
        for line in git(*command).splitlines():
            candidate = line.strip()
            if candidate and (root / candidate).exists():
                paths.add(PurePosixPath(candidate).as_posix())
    return paths


def matches_any(path: str, patterns: list[str]) -> bool:
    pure_path = PurePosixPath(path)
    return any(fnmatch.fnmatch(path, pattern) or pure_path.match(pattern) for pattern in patterns)


def is_code(path: str, config: dict[str, list[str]]) -> bool:
    if matches_any(path, config["exclude_patterns"]):
        return False
    pure_path = PurePosixPath(path)
    return pure_path.name in config["code_filenames"] or pure_path.suffix in config["code_extensions"]


def nearest_module_readme(root: Path, code_path: str) -> Path | None:
    relative = PurePosixPath(code_path)
    parent = (root / relative.parent).resolve()

    if len(relative.parts) == 1:
        candidate = root / "README.md"
        return candidate if candidate.exists() else None

    while parent != root:
        candidate = parent / "README.md"
        if candidate.exists():
            return candidate
        if root not in parent.parents:
            break
        parent = parent.parent
    return None


def check_documentation(root: Path, base: str | None) -> int:
    config = load_config(root)
    changed = changed_files(root, base)
    code_paths = sorted(path for path in changed if is_code(path, config))

    if not code_paths:
        print("No documentation-sensitive code changes found.")
        return 0

    violations: list[tuple[str, str]] = []
    for code_path in code_paths:
        readme = nearest_module_readme(root, code_path)
        if readme is None:
            violations.append((code_path, "no README.md found in the logical module"))
            continue
        readme_path = readme.relative_to(root).as_posix()
        if readme_path not in changed:
            violations.append((code_path, f"{readme_path} was not updated"))

    if violations:
        print("Documentation check failed:")
        for code_path, reason in violations:
            print(f"  - {code_path}: {reason}")
        print("Update the nearest module README or adjust docs/docs-config.json for a true exception.")
        return 1

    print(f"Documentation check passed for {len(code_paths)} code file(s).")
    return 0


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)

    index_parser = subparsers.add_parser("index", help="generate the common documentation index")
    index_parser.add_argument("--check", action="store_true", help="check without changing files")

    check_parser = subparsers.add_parser("check", help="verify code changes update module docs")
    check_parser.add_argument("--base", help="also include the diff from this Git ref to HEAD")
    return parser.parse_args()


def main() -> int:
    try:
        args = parse_args()
        root = repository_root()
        if args.command == "index":
            return update_indexes(root, args.check)
        return check_documentation(root, args.base)
    except (OSError, RuntimeError, json.JSONDecodeError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
