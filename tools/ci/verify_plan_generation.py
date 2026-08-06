#!/usr/bin/env python3
"""Verify plan generation is idempotent: two runs must not change plans/."""

from __future__ import annotations

import argparse
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
GEN_SCRIPT = ROOT / "plans" / "scripts" / "enrich_and_generate.py"


def run(cmd: list[str], *, cwd: Path = ROOT) -> None:
    proc = subprocess.run(cmd, cwd=cwd, check=False)
    if proc.returncode != 0:
        raise SystemExit(proc.returncode)


def git_diff_quiet(path: str = "plans/") -> bool:
    proc = subprocess.run(
        ["git", "diff", "--quiet", "--exit-code", path],
        cwd=ROOT,
        check=False,
    )
    return proc.returncode == 0


def _git_diff(path: str = "plans/") -> None:
    """Print git diff for diagnostics."""
    proc = subprocess.run(
        ["git", "diff", "--stat", path],
        cwd=ROOT,
        check=False,
    )
    proc = subprocess.run(
        ["git", "diff", path],
        cwd=ROOT,
        check=False,
    )


def invoke_plan_generation() -> None:
    if not GEN_SCRIPT.is_file():
        raise SystemExit(f"missing plan generator: {GEN_SCRIPT}")
    run([sys.executable, str(GEN_SCRIPT)])


def verify_idempotency() -> None:
    if not git_diff_quiet():
        _git_diff("plans/")
        raise SystemExit(
            "plans/ has uncommitted changes; commit or discard before idempotency check"
        )

    print("plan-generation idempotency: first run")
    invoke_plan_generation()
    if not git_diff_quiet():
        _git_diff("plans/")
        raise SystemExit("plans/ changed after first generation run")

    print("plan-generation idempotency: second run")
    invoke_plan_generation()
    if not git_diff_quiet():
        _git_diff("plans/")
        raise SystemExit(
            "plans/ changed after second generation run (not idempotent)"
        )

    print("ok: plan generation is idempotent")


def self_check() -> None:
    missing: list[str] = []
    for tool in ("git", "python3"):
        if shutil.which(tool) is None:
            missing.append(tool)
    if not GEN_SCRIPT.is_file():
        missing.append(str(GEN_SCRIPT))
    if missing:
        raise SystemExit(f"self-check failed: missing {', '.join(missing)}")
    print("ok: verify_plan_generation self-check")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--self-check",
        action="store_true",
        help="verify required tools and paths exist",
    )
    args = parser.parse_args()
    if args.self_check:
        self_check()
        return
    verify_idempotency()


if __name__ == "__main__":
    main()
