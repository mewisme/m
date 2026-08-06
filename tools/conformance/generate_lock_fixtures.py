#!/usr/bin/env python3
"""Regenerate committed lock bridge conformance fixtures from pinned pnpm binaries.

Reads family sources from fixtures/locks/sources/pnpm/<family>/ and writes
fixtures/locks/generated/pnpm-{9,10,11}/<family>/ with honest metadata.json.
Use --generate to run isolated temp homes with exact pnpm@X.Y.Z via corepack.

Replaces: generate-lock-fixtures.ps1
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
import uuid
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SOURCES_DIR = ROOT / "fixtures" / "locks" / "sources" / "pnpm"
GENERATED_DIR = ROOT / "fixtures" / "locks" / "generated"
VERSIONS_ENV = Path(__file__).resolve().parent / "pnpm-versions.env"
DIGEST_TOOL = Path(__file__).resolve().parent / "fixturemeta" / "cmd"
REGISTRY_URL = "https://registry.npmjs.org"

DEFAULT_FAMILIES = [
    "basic", "transitive", "optional", "peer-context", "multi-version",
    "scoped", "workspace", "catalog", "override", "platform", "importer-meta",
    "alias", "alias-peer", "patch", "binary",
]
DEFAULT_MAJORS = [9, 10, 11]


def load_pnpm_versions() -> dict[str, str]:
    versions: dict[str, str] = {}
    if not VERSIONS_ENV.is_file():
        raise SystemExit(f"missing {VERSIONS_ENV}")
    for line in VERSIONS_ENV.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" in line:
            k, v = line.split("=", 1)
            versions[k.strip()] = v.strip()
    return versions


def fixture_digest(mode: str, path: Path) -> str:
    env = os.environ.copy()
    env["CGO_ENABLED"] = "0"
    result = subprocess.run(
        ["go", "run", str(DIGEST_TOOL), mode, str(path)],
        capture_output=True, text=True, env=env, cwd=ROOT,
    )
    if result.returncode != 0:
        raise SystemExit(f"digest {mode} {path} failed: {result.stderr}")
    return result.stdout.strip()


def sha256_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def get_workspace_manifest_hashes(dir_path: Path) -> dict[str, str] | None:
    packages = dir_path / "packages"
    if not packages.is_dir():
        return None
    out: dict[str, str] = {}
    for pkg_json in packages.rglob("package.json"):
        rel = str(pkg_json.relative_to(dir_path)).replace("\\", "/")
        out[rel] = fixture_digest("file", pkg_json)
    return out if out else None


def get_patch_file_hashes(dir_path: Path) -> dict[str, str] | None:
    patches = dir_path / "patches"
    if not patches.is_dir():
        return None
    out: dict[str, str] = {}
    for patch_file in patches.rglob("*.patch"):
        rel = str(patch_file.relative_to(dir_path)).replace("\\", "/")
        out[rel] = fixture_digest("file", patch_file)
    return out if out else None


def write_metadata(dest: Path, src: Path, major: int, family: str,
                   pnpm_version: str, executable_path: str,
                   executable_args: list[str], invocation_id: str) -> None:
    lockfile = dest / "pnpm-lock.yaml"
    pkg_json = dest / "package.json"
    if not lockfile.is_file():
        raise SystemExit(f"missing lock at {lockfile}")
    if not pkg_json.is_file():
        raise SystemExit(f"missing package.json at {pkg_json}")

    lock_hash = fixture_digest("file", lockfile)
    pkg_hash = fixture_digest("file", pkg_json)
    source_digest = fixture_digest("source-tree", src)

    ws_yaml = dest / "pnpm-workspace.yaml"
    ws_hash = sha256_file(ws_yaml) if ws_yaml.is_file() else None

    ws_manifests = get_workspace_manifest_hashes(dest)
    patch_hashes = get_patch_file_hashes(dest)

    cmd = f"corepack prepare pnpm@{pnpm_version} --activate; pnpm install --lockfile-only --ignore-scripts (family={family})"

    meta: dict = {
        "producer": "pnpm",
        "producerVersion": pnpm_version,
        "producerMajor": major,
        "family": family,
        "node": subprocess.run(["node", "-v"], capture_output=True, text=True).stdout.strip() or "",
        "os": sys.platform,
        "arch": os.uname().machine if hasattr(os, "uname") else "",
        "executablePath": executable_path,
        "executableArgs": executable_args,
        "registry": REGISTRY_URL,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "classification": "generated",
        "lockfileVersion": "9.0",
        "lockfileSha256": lock_hash,
        "packageJsonSha256": pkg_hash,
        "sourceTreeDigest": source_digest,
        "invocationId": invocation_id,
        "isolatedHomePolicy": "temp-home-and-pnpm-store-per-family",
        "command": cmd,
        "confidence": "certain",
        "generationSignals": [
            "lockfileVersion=9.0",
            f"family={family}",
            f"invocationId={invocation_id}",
        ],
    }
    if ws_hash:
        meta["workspaceYamlSha256"] = ws_hash
    if ws_manifests:
        meta["workspaceManifestSha256"] = ws_manifests
    if patch_hashes:
        meta["patchFileSha256"] = patch_hashes

    meta_path = dest / "metadata.json"
    meta_path.write_text(json.dumps(meta, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")


def copy_family_source(src: Path, dest: Path) -> None:
    dest.mkdir(parents=True, exist_ok=True)
    for item in src.rglob("*"):
        if item.is_dir():
            continue
        rel = item.relative_to(src)
        rel_str = str(rel)
        if rel_str.startswith(".home") or rel_str.startswith(".pnpm-store"):
            continue
        out = dest / rel
        out.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(item, out)


def invoke_pnpm_lock_only(work_dir: Path, pnpm_version: str) -> tuple[str, list[str]]:
    env = os.environ.copy()
    env["COREPACK_ENABLE_DOWNLOAD_PROMPT"] = "0"
    subprocess.run(["corepack", "enable"], capture_output=True, env=env)
    subprocess.run(
        ["corepack", "prepare", f"pnpm@{pnpm_version}", "--activate"],
        capture_output=True, env=env, check=True,
    )
    pnpm_path = shutil.which("pnpm")
    if not pnpm_path:
        raise SystemExit("pnpm not found after corepack prepare")
    args = ["install", "--lockfile-only", "--ignore-scripts"]
    result = subprocess.run(
        [pnpm_path] + args, cwd=work_dir, capture_output=True, env=env,
    )
    if result.returncode != 0:
        raise SystemExit(f"pnpm install failed in {work_dir} (exit {result.returncode})")
    return pnpm_path, args


def generate_pnpm_fixtures(majors: list[int], families: list[str], generate: bool,
                           invocation_id: str) -> None:
    versions = load_pnpm_versions()

    for major in majors:
        ver_key = f"PNPM{major}_VERSION"
        if ver_key not in versions:
            raise SystemExit(f"missing {ver_key} in pnpm-versions.env")
        ver = versions[ver_key]

        for family in families:
            src = SOURCES_DIR / family
            if not src.is_dir():
                raise SystemExit(f"missing source family {src}")

            dest = GENERATED_DIR / f"pnpm-{major}" / family

            if generate:
                work = Path(tempfile.mkdtemp(prefix=f"mew-lock-fix-{major}-{family}-"))
                home_dir = work / ".home"
                store_dir = work / ".pnpm-store"
                home_dir.mkdir(parents=True)
                store_dir.mkdir(parents=True)

                saved_home = os.environ.get("HOME")
                saved_userprofile = os.environ.get("USERPROFILE")
                saved_pnpm_home = os.environ.get("PNPM_HOME")

                os.environ["HOME"] = str(home_dir)
                os.environ["USERPROFILE"] = str(home_dir)
                os.environ["PNPM_HOME"] = str(home_dir / "pnpm")

                try:
                    copy_family_source(src, work)
                    exe_path, exe_args = invoke_pnpm_lock_only(work, ver)
                    lockfile = work / "pnpm-lock.yaml"
                    if not lockfile.is_file():
                        raise SystemExit(f"pnpm did not write lockfile for {family} major={major}")
                    copy_family_source(work, dest)
                    write_metadata(dest, src, major, family, ver,
                                   exe_path, exe_args, invocation_id)
                finally:
                    if saved_home is not None:
                        os.environ["HOME"] = saved_home
                    else:
                        os.environ.pop("HOME", None)
                    if saved_userprofile is not None:
                        os.environ["USERPROFILE"] = saved_userprofile
                    if saved_pnpm_home is not None:
                        os.environ["PNPM_HOME"] = saved_pnpm_home
                    shutil.rmtree(work, ignore_errors=True)

                print(f"ok: pnpm-{major}/{family}")
            else:
                if not (dest / "pnpm-lock.yaml").is_file():
                    print(f"skip {dest} — no committed lock; run with --generate", file=sys.stderr)
                    continue
                if not (dest / "metadata.json").is_file():
                    print(f"skip {dest} — no metadata.json; run with --generate", file=sys.stderr)
                    continue
                print(f"ok: pnpm-{major}/{family} (verify-only)")


def generate_nub_fixtures(invocation_id: str) -> None:
    nub_map = {
        "nub-basic": "basic",
        "nub-transitive": "transitive",
        "nub-workspace": "workspace",
        "nub-catalog": "catalog",
        "nub-peer": "peer-context",
        "nub-optional": "optional",
    }

    for nub_name, family in nub_map.items():
        pnpm_dir = GENERATED_DIR / "pnpm-9" / family
        nub_dest = GENERATED_DIR / nub_name

        source_lock = pnpm_dir / "pnpm-lock.yaml"
        if not source_lock.is_file():
            print(f"skip {nub_name} — missing pnpm-9/{family}", file=sys.stderr)
            continue

        source_lock_hash = fixture_digest("file", source_lock)
        nub_dest.mkdir(parents=True, exist_ok=True)

        # Copy package.json and workspace config
        shutil.copy2(pnpm_dir / "package.json", nub_dest / "package.json")
        ws_yaml_src = pnpm_dir / "pnpm-workspace.yaml"
        if ws_yaml_src.is_file():
            shutil.copy2(ws_yaml_src, nub_dest / "pnpm-workspace.yaml")

        packages_src = pnpm_dir / "packages"
        packages_dst = nub_dest / "packages"
        if packages_src.is_dir():
            if packages_dst.exists():
                shutil.rmtree(packages_dst)
            shutil.copytree(packages_src, packages_dst)

        # Append nubVersion marker
        lock_content = source_lock.read_text(encoding="utf-8")
        if "nubVersion:" not in lock_content:
            lock_content = lock_content.rstrip() + "\nnubVersion: \"1.0.0\"\n"
        nub_lock_path = nub_dest / "nub.lock"
        nub_lock_path.write_text(lock_content, encoding="utf-8")

        nub_hash = fixture_digest("file", nub_lock_path)
        pkg_hash = fixture_digest("file", nub_dest / "package.json")
        src_rel = f"fixtures/locks/generated/pnpm-9/{family}"
        deriv_cmd = f"derived from {src_rel} pnpm-lock.yaml; append nubVersion: 1.0.0"
        ws_hash = None
        ws_yaml = nub_dest / "pnpm-workspace.yaml"
        if ws_yaml.is_file():
            ws_hash = sha256_file(ws_yaml)
        ws_manifests = get_workspace_manifest_hashes(nub_dest)
        source_tree_digest = fixture_digest("source-tree", SOURCES_DIR / family)

        meta = {
            "producer": "nub",
            "producerVersion": "pnpm-9-shaped",
            "producerMajor": 9,
            "family": nub_name,
            "node": subprocess.run(["node", "-v"], capture_output=True, text=True).stdout.strip(),
            "os": sys.platform,
            "arch": os.uname().machine if hasattr(os, "uname") else "",
            "executablePath": "derived",
            "executableArgs": ["nub.lock", "from", "pnpm-9", family],
            "registry": REGISTRY_URL,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "classification": "derived",
            "lockfileVersion": "9.0",
            "lockfileSha256": nub_hash,
            "packageJsonSha256": pkg_hash,
            "sourceFixture": src_rel,
            "sourceLockSha256": source_lock_hash,
            "derivationCommand": deriv_cmd,
            "sourceTreeDigest": source_tree_digest,
            "invocationId": invocation_id,
            "isolatedHomePolicy": "temp-home-and-pnpm-store-per-family",
            "command": deriv_cmd,
            "confidence": "manual",
            "generationSignals": [
                "pnpm-v9-shaped",
                "nub.lock",
                f"derived-from=pnpm-9/{family}",
                f"invocationId={invocation_id}",
            ],
        }
        if ws_hash:
            meta["workspaceYamlSha256"] = ws_hash
        if ws_manifests:
            meta["workspaceManifestSha256"] = ws_manifests

        (nub_dest / "metadata.json").write_text(
            json.dumps(meta, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
        print(f"ok: {nub_name}")


def main() -> None:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--generate", action="store_true",
                   help="Run pnpm install in isolated temp homes (requires corepack)")
    p.add_argument("--families", nargs="*", default=DEFAULT_FAMILIES,
                   help="Fixture families to process")
    p.add_argument("--majors", type=int, nargs="*", default=DEFAULT_MAJORS,
                   help="pnpm major versions")
    args = p.parse_args()

    invocation_id = str(uuid.uuid4())

    generate_pnpm_fixtures(args.majors, args.families, args.generate, invocation_id)

    if args.generate:
        generate_nub_fixtures(invocation_id)

    print(f"done: fixtures/locks/generated refreshed (invocationId={invocation_id})")


if __name__ == "__main__":
    main()
