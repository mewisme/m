#!/usr/bin/env python3
"""Validate CycloneDX SBOM golden structure (no external schema CLI required)."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
DEFAULT_SBOM = ROOT / "fixtures" / "sbom" / "medium-graph-cyclonedx-golden.json"


def validate(path: Path) -> None:
    with open(path, "r", encoding="utf-8") as f:
        doc = json.load(f)

    if doc.get("bomFormat") != "CycloneDX":
        raise SystemExit("bomFormat must be CycloneDX")
    if doc.get("specVersion") != "1.5":
        raise SystemExit("specVersion must be 1.5")

    metadata = doc.get("metadata", {})
    component = metadata.get("component", {})
    if not component.get("bom-ref"):
        raise SystemExit("metadata.component.bom-ref required")

    components = doc.get("components", [])
    if not components:
        raise SystemExit("components required")

    dependencies = doc.get("dependencies", [])
    if not dependencies:
        raise SystemExit("dependencies required")

    refs: set[str] = set()
    for c in components:
        bref = c.get("bom-ref")
        if not bref:
            raise SystemExit(f"component missing bom-ref: {c.get('name', '?')}")
        refs.add(bref)
    refs.add(component["bom-ref"])

    for dep in dependencies:
        dref = dep.get("ref")
        if not dref:
            raise SystemExit("dependency missing ref")
        if dref not in refs:
            raise SystemExit(f"unknown dependency ref: {dref}")
        for to_ref in dep.get("dependsOn", []):
            if to_ref not in refs:
                raise SystemExit(f"unknown dependsOn ref: {to_ref}")

    print(f"OK: {path}")


def main() -> None:
    p = argparse.ArgumentParser(description="Validate CycloneDX SBOM golden structure.")
    p.add_argument("path", nargs="?", type=Path, default=DEFAULT_SBOM,
                   help="Path to SBOM JSON file")
    args = p.parse_args()
    validate(args.path)


if __name__ == "__main__":
    main()
