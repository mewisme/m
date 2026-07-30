#!/usr/bin/env python3
"""Run short fuzz smoke on packages that declare Fuzz* tests."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FUZZ_RE = re.compile(r"^func Fuzz", re.MULTILINE)


def main() -> None:
    proc = subprocess.run(
        ["go", "list", "./..."],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=True,
    )
    found = False
    for pkg in proc.stdout.splitlines():
        pkg = pkg.strip()
        if not pkg:
            continue
        dir_proc = subprocess.run(
            ["go", "list", "-f", "{{.Dir}}", pkg],
            cwd=ROOT,
            capture_output=True,
            text=True,
            check=True,
        )
        pkg_dir = Path(dir_proc.stdout.strip())
        has_fuzz = False
        for test_file in pkg_dir.glob("*_test.go"):
            if FUZZ_RE.search(test_file.read_text(encoding="utf-8", errors="replace")):
                has_fuzz = True
                break
        if not has_fuzz:
            continue
        found = True
        print(f"fuzz-smoke: {pkg}")
        code = subprocess.run(
            ["go", "test", pkg, "-fuzz=.", "-fuzztime=1s", "-count=1"],
            cwd=ROOT,
            check=False,
        ).returncode
        if code != 0:
            raise SystemExit(code)
    if not found:
        print("fuzz-smoke: no Fuzz* targets; ok")


if __name__ == "__main__":
    main()
