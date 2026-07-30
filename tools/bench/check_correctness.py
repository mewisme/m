#!/usr/bin/env python3
"""Run install bench and validate correctness fields."""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
OUT_PATH = ROOT / "bench-result.json"
MIN_SAMPLES = 7


def run_bench(mode: str) -> dict:
    env = os.environ.copy()
    env["CGO_ENABLED"] = "0"
    proc = subprocess.run(
        ["go", "run", "./cmd/m", "bench", "install", f"--{mode}", "--json"],
        cwd=ROOT,
        env=env,
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        raise SystemExit(f"bench install failed: {proc.stdout}{proc.stderr}")
    line = proc.stdout.strip().splitlines()[-1].strip()
    return json.loads(line)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--mode",
        choices=("cold", "warm"),
        default="warm",
        help="bench install mode (default: warm)",
    )
    args = parser.parse_args()

    result = run_bench(args.mode)
    for key in ("case", "mode", "fixtureDigest", "medianMs"):
        if not result.get(key):
            raise SystemExit(f"bench JSON missing {key}")
    samples = result.get("samples") or []
    if len(samples) < MIN_SAMPLES:
        raise SystemExit(f"bench samples={len(samples)} require >= {MIN_SAMPLES}")

    artifact = {
        "checkedAt": datetime.now(timezone.utc).isoformat(),
        "mode": args.mode,
        "result": result,
    }
    OUT_PATH.write_text(
        json.dumps(artifact, indent=2) + "\n", encoding="utf-8"
    )
    print(
        "ok: bench correctness "
        f"case={result['case']} samples={len(samples)} medianMs={result['medianMs']}"
    )


if __name__ == "__main__":
    main()
