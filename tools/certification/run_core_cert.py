#!/usr/bin/env python3
"""Run core certification steps from core-manifest.json."""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
MANIFEST_PATH = Path(__file__).resolve().parent / "core-manifest.json"

VALID_TARGETS = (
    "core-cert-fast",
    "core-cert",
    "core-cert-security",
    "core-cert-crash",
    "core-cert-performance",
)


def tool_available(name: str) -> bool:
    return shutil.which(name) is not None


def run_step(command: str, *, shell: str | None, env: dict[str, str]) -> int:
    merged = os.environ.copy()
    merged.update(env)
    if shell == "pwsh":
        proc = subprocess.run(
            ["pwsh", "-NoProfile", "-Command", command],
            cwd=ROOT,
            env=merged,
            check=False,
        )
        return proc.returncode
    proc = subprocess.run(command, cwd=ROOT, env=merged, shell=True, check=False)
    return proc.returncode


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "target",
        choices=VALID_TARGETS,
        help="certification target from core-manifest.json",
    )
    args = parser.parse_args()

    manifest = json.loads(MANIFEST_PATH.read_text(encoding="utf-8"))
    targets = manifest.get("targets") or {}
    steps = manifest.get("steps") or {}
    if args.target not in targets:
        raise SystemExit(f"unknown target: {args.target}")

    step_ids = targets[args.target].get("steps") or []
    print(f"core-cert target={args.target} steps={len(step_ids)}")

    failures = 0
    for step_id in step_ids:
        step = steps.get(step_id)
        if not step:
            raise SystemExit(f"manifest missing step {step_id}")

        if step.get("requiresTools"):
            missing: list[str] = []
            skip = False
            for tool in step["requiresTools"]:
                if tool == "govulncheck" and not tool_available(tool):
                    skip = True
                    break
                if tool in ("node", "pnpm") and not tool_available(tool):
                    missing.append(tool)
            if skip:
                print(f"skip {step['id']}: required tool not installed")
                continue
            if missing and step.get("blocking"):
                raise SystemExit(
                    f"step {step['id']} requires tools: {', '.join(missing)}"
                )
            if missing and not step.get("blocking"):
                print(f"skip {step['id']}: missing tools {', '.join(missing)}")
                continue

        step_env = {str(k): str(v) for k, v in (step.get("env") or {}).items()}
        print(f"==> {step['id']}: {step['command']}")
        code = run_step(step["command"], shell=step.get("shell"), env=step_env)
        if code != 0:
            if step.get("blocking"):
                raise SystemExit(f"step {step['id']} failed with exit {code}")
            print(f"WARN: non-blocking step {step['id']} failed with exit {code}")
            failures += 1

    if failures:
        print(f"completed with {failures} non-blocking failure(s)")
    print(f"ok: {args.target}")


if __name__ == "__main__":
    main()
