#!/usr/bin/env python3
"""Smoke checks for MewJS development install scripts."""

from __future__ import annotations

import argparse
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
INSTALL_SH = ROOT / "scripts" / "install-dev.sh"
INSTALL_PS1 = ROOT / "scripts" / "install-dev.ps1"
UNINSTALL_SH = ROOT / "scripts" / "uninstall-dev.sh"


def run(cmd: list[str], *, cwd: Path = ROOT, env: dict[str, str] | None = None) -> None:
    merged = os.environ.copy()
    if env:
        merged.update(env)
    proc = subprocess.run(cmd, cwd=cwd, env=merged, check=False)
    if proc.returncode != 0:
        raise SystemExit(proc.returncode)


def resolve_bash() -> str:
    bash = shutil.which("bash")
    if bash:
        return bash
    candidates = [
        Path(r"C:\Program Files\Git\bin\bash.exe"),
        Path(r"C:\Program Files (x86)\Git\bin\bash.exe"),
    ]
    for candidate in candidates:
        if candidate.is_file():
            return str(candidate)
    raise SystemExit("smoke: bash not found")


def self_check() -> None:
    missing: list[str] = []
    for path in (INSTALL_SH, INSTALL_PS1, UNINSTALL_SH):
        if not path.is_file():
            missing.append(str(path))
    try:
        resolve_bash()
    except SystemExit:
        missing.append("bash")
    if shutil.which("pwsh") is None:
        missing.append("pwsh")
    if missing:
        raise SystemExit(f"self-check failed: missing {', '.join(missing)}")
    print("ok: devinstall smoke self-check")


def smoke_build_only() -> None:
    with tempfile.TemporaryDirectory(prefix="mew-devinstall-") as tmp:
        home = Path(tmp) / "home"
        home.mkdir()
        env = {
            "HOME": str(home),
            "XDG_DATA_HOME": str(home / ".local" / "share"),
            "CGO_ENABLED": "0",
        }
        if os.name == "nt":
            env["LOCALAPPDATA"] = str(home / "AppData" / "Local")
            print("smoke: install-dev.ps1 -BuildOnly")
            run(["pwsh", "-NoProfile", "-File", str(INSTALL_PS1), "-BuildOnly"], env=env)
            m_bin = ROOT / "bin" / "m.exe"
        else:
            bash = resolve_bash()
            print("smoke: install-dev.sh --build-only")
            run([bash, str(INSTALL_SH), "--build-only"], env=env)
            m_bin = ROOT / "bin" / "m"
        if not m_bin.is_file():
            raise SystemExit(f"missing built binary: {m_bin}")
        run([str(m_bin), "version"], env=env)
        print("ok: devinstall smoke build-only")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--self-check", action="store_true")
    parser.add_argument("--build-only", action="store_true", help="run isolated build-only smoke")
    args = parser.parse_args()
    if args.self_check:
        self_check()
        return
    if args.build_only:
        smoke_build_only()
        return
    self_check()
    smoke_build_only()


if __name__ == "__main__":
    main()
