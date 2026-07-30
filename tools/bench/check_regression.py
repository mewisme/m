#!/usr/bin/env python3
"""Compare install bench results against platform baseline."""

from __future__ import annotations

import argparse
import json
import os
import platform
import subprocess
import sys
from datetime import date, datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
BASELINE_PATH = ROOT / "benchmarks" / "install-baseline.json"
WAIVERS_PATH = ROOT / "benchmarks" / "waivers.json"
OUT_PATH = ROOT / "bench-result.json"
NOISE_BUDGET = 0.10
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


def runner_class(result: dict) -> str:
    if result.get("runnerClass"):
        return str(result["runnerClass"])
    if os.environ.get("MEW_BENCH_RUNNER_CLASS"):
        return os.environ["MEW_BENCH_RUNNER_CLASS"]
    if os.environ.get("GITHUB_ACTIONS") == "true":
        runner = (os.environ.get("RUNNER_OS") or "unknown").lower()
        return f"github-actions-{runner}"
    system = platform.system().lower()
    if system == "windows":
        return "local-windows"
    if system == "linux":
        return "local-linux"
    if system == "darwin":
        return "local-darwin"
    return "local-unknown"


def load_waivers() -> list[dict]:
    if not WAIVERS_PATH.is_file():
        return []
    data = json.loads(WAIVERS_PATH.read_text(encoding="utf-8"))
    return list(data.get("waivers") or [])


def match_waiver(waivers: list[dict], result: dict, runner: str) -> dict | None:
    today = date.today()
    for waiver in waivers:
        if waiver.get("case") != result.get("case"):
            continue
        if waiver.get("benchmarkMode") and waiver["benchmarkMode"] != result.get("mode"):
            continue
        if waiver.get("os") and waiver["os"] != result.get("os"):
            continue
        if waiver.get("arch") and waiver["arch"] != result.get("arch"):
            continue
        if waiver.get("runnerClass") and waiver["runnerClass"] != runner:
            continue
        expires = waiver.get("expires")
        if expires:
            exp = date.fromisoformat(str(expires)[:10])
            if today > exp:
                continue
        return waiver
    return None


def find_baseline(cases: list[dict], case_name: str, result: dict, mode: str, runner: str) -> dict | None:
    for entry in cases:
        if entry.get("name") != case_name:
            continue
        if entry.get("os") != result.get("os"):
            continue
        if entry.get("arch") != result.get("arch"):
            continue
        if entry.get("benchmarkMode") != mode:
            continue
        rc = entry.get("runnerClass")
        if rc and rc != runner:
            continue
        return entry
    return None


def write_artifact(payload: dict) -> None:
    OUT_PATH.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--mode",
        choices=("cold", "warm"),
        default="warm",
        help="bench install mode (default: warm)",
    )
    args = parser.parse_args()

    if os.environ.get("BENCH_WAIVER") == "1":
        print(
            "warning: BENCH_WAIVER=1 is deprecated; use benchmarks/waivers.json structured waivers",
            file=sys.stderr,
        )

    if not BASELINE_PATH.is_file():
        raise SystemExit(f"missing baseline file: {BASELINE_PATH}")

    result = run_bench(args.mode)
    runner = runner_class(result)
    case_name = f"medium-graph-{args.mode}"
    samples = result.get("samples") or []
    if len(samples) < MIN_SAMPLES:
        raise SystemExit(f"bench samples={len(samples)} require >= {MIN_SAMPLES}")

    baseline = json.loads(BASELINE_PATH.read_text(encoding="utf-8"))
    entry = find_baseline(baseline.get("cases") or [], case_name, result, args.mode, runner)
    waivers = load_waivers()

    if entry is None:
        waiver = match_waiver(waivers, result, runner)
        if waiver:
            print(
                "WARN: no platform baseline for "
                f"case={case_name} os={result.get('os')} arch={result.get('arch')} "
                f"runner={runner}; waiver owner={waiver.get('owner')} "
                f"reason={waiver.get('reason')}"
            )
            write_artifact(
                {
                    "checkedAt": datetime.now(timezone.utc).isoformat(),
                    "mode": args.mode,
                    "runnerClass": runner,
                    "waived": True,
                    "waiver": waiver,
                    "result": result,
                }
            )
            return
        raise SystemExit(
            "no baseline case for "
            f"case={case_name} os={result.get('os')} arch={result.get('arch')} "
            f"runner={runner} mode={args.mode}"
        )

    base_digest = entry.get("fixtureDigest")
    cur_digest = result.get("fixtureDigest")
    if base_digest and cur_digest and base_digest != cur_digest:
        raise SystemExit(
            f"fixtureDigest mismatch baseline={base_digest} current={cur_digest}"
        )

    median_baseline = float(entry["totalMsMedian"])
    p95_baseline = float(entry.get("totalMsP95", median_baseline))
    median_limit = median_baseline * (1.0 + NOISE_BUDGET)
    p95_limit = p95_baseline * (1.0 + NOISE_BUDGET)
    median = float(result["medianMs"])
    p95 = float(result.get("p95Ms", result.get("totalMs", median)))

    print(
        f"case={case_name} runner={runner} samples={len(samples)} "
        f"medianMs={median} p95Ms={p95} medianLimit={median_limit} p95Limit={p95_limit}"
    )

    median_regression = median > median_limit
    p95_regression = p95 > p95_limit
    if median_regression or p95_regression:
        parts: list[str] = []
        pct = int(NOISE_BUDGET * 100)
        if median_regression:
            parts.append(
                f"medianMs {median} exceeds limit {median_limit} "
                f"(+{pct}% of baseline {median_baseline})"
            )
        if p95_regression:
            parts.append(
                f"p95Ms {p95} exceeds limit {p95_limit} "
                f"(+{pct}% of baseline {p95_baseline})"
            )
        message = "bench regression: " + "; ".join(parts)
        waiver = match_waiver(waivers, result, runner)
        if waiver:
            print(f"WARN: {message}; structured waiver owner={waiver.get('owner')}")
            write_artifact(
                {
                    "checkedAt": datetime.now(timezone.utc).isoformat(),
                    "mode": args.mode,
                    "runnerClass": runner,
                    "waived": True,
                    "waiver": waiver,
                    "regression": message,
                    "result": result,
                }
            )
            return
        if os.environ.get("BENCH_WAIVER") == "1":
            print(f"warning: {message} but BENCH_WAIVER=1 is deprecated", file=sys.stderr)
            return
        raise SystemExit(message)

    write_artifact(
        {
            "checkedAt": datetime.now(timezone.utc).isoformat(),
            "mode": args.mode,
            "runnerClass": runner,
            "baseline": entry,
            "result": result,
        }
    )
    print("ok: within baseline median and p95")


if __name__ == "__main__":
    main()
