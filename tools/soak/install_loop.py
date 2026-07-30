#!/usr/bin/env python3
"""Repeated install bench soak loop."""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--count", type=int, default=100, help="iterations")
    parser.add_argument(
        "--mode",
        choices=("cold", "warm"),
        default="cold",
        help="bench install mode",
    )
    parser.add_argument("--project", default="", help="fixture project name")
    args = parser.parse_args()

    flag = f"--{args.mode}"
    cmd = ["go", "run", "./cmd/m", "bench", "install", flag, "--json"]
    if args.project:
        cmd.extend(["--fixture", args.project])

    env = os.environ.copy()
    env["CGO_ENABLED"] = "0"
    if args.project and "workspace" in args.project:
        env["MEW_EXPERIMENTAL_WORKSPACES"] = "1"
        env["MEW_EXPERIMENTAL_ISOLATED_LINKER"] = "1"

    label = args.project or "default"
    for i in range(1, args.count + 1):
        print(f"install loop {i}/{args.count} mode={args.mode} project={label}")
        proc = subprocess.run(cmd, cwd=ROOT, env=env, check=False)
        if proc.returncode != 0:
            raise SystemExit(f"bench install failed on iteration {i}")

    print(f"ok: {args.count} install iterations completed project={label}")


if __name__ == "__main__":
    main()
