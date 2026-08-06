#!/usr/bin/env python3
"""Tests for tools/update-runtime-assets.py — run with:
  python3 -m pytest tools/test_update-runtime-assets.py -v
  python3 tools/test_update-runtime-assets.py  # also works
"""

from __future__ import annotations

import hashlib
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TOOL = ROOT / "tools" / "update-runtime-assets.py"


def run_tool(*args: str, cwd: Path | None = None) -> subprocess.CompletedProcess:
    """Run the manifest tool and return CompletedProcess."""
    if cwd is None:
        cwd = ROOT
    return subprocess.run(
        [sys.executable, str(TOOL), *args],
        cwd=cwd,
        capture_output=True,
        text=True,
    )


def make_manifest(
    assets_dir: Path,
    manifest_path: Path,
    assets: list[dict] | None = None,
    schema_version: int = 2,
    bundle_version: str = "4",
) -> None:
    """Write a manifest.json for testing."""
    data = {
        "schemaVersion": schema_version,
        "bundleVersion": bundle_version,
        "assets": assets or [],
    }
    manifest_path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")


def make_asset(dir_path: Path, name: str, content: bytes) -> Path:
    """Create a file in the assets directory and return its path."""
    p = dir_path / name
    p.write_bytes(content)
    return p


def sha256_hex(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


class ToolExistsTests(unittest.TestCase):
    def test_tool_found(self) -> None:
        self.assertTrue(TOOL.is_file(), f"Tool not found at {TOOL}")

    def test_tool_runnable(self) -> None:
        result = run_tool("--help")
        self.assertEqual(result.returncode, 0)


class CheckUnchangedTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.assets_dir = Path(self.tmp.name) / "assets"
        self.assets_dir.mkdir()
        self.manifest = Path(self.tmp.name) / "manifest.json"

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def test_unchanged_passes_check(self) -> None:
        content = b"console.log(1);"
        make_asset(self.assets_dir, "test.mjs", content)
        sz, sha = len(content), sha256_hex(content)
        make_manifest(
            self.assets_dir,
            self.manifest,
            [{"name": "test.mjs", "path": "test.mjs", "role": "loader-support",
              "moduleType": "esm", "size": sz, "sha256": sha}],
        )
        result = run_tool("--check", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")

    def test_stale_fails_check(self) -> None:
        content = b"console.log(2);"
        make_asset(self.assets_dir, "test.mjs", content)
        make_manifest(
            self.assets_dir,
            self.manifest,
            [{"name": "test.mjs", "path": "test.mjs", "role": "loader-support",
              "moduleType": "esm", "size": 999, "sha256": "a" * 64}],
        )
        result = run_tool("--check", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertNotEqual(result.returncode, 0)

    def test_write_updates_stale_metadata(self) -> None:
        content = b"console.log(3);"
        make_asset(self.assets_dir, "test.mjs", content)
        sz, sha = len(content), sha256_hex(content)
        make_manifest(
            self.assets_dir,
            self.manifest,
            [{"name": "test.mjs", "path": "test.mjs", "role": "loader-support",
              "moduleType": "esm", "size": 999, "sha256": "b" * 64}],
        )
        result = run_tool("--write", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")

        updated = json.loads(self.manifest.read_text(encoding="utf-8"))
        entry = updated["assets"][0]
        self.assertEqual(entry["size"], sz)
        self.assertEqual(entry["sha256"], sha)


class AddRemoveTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.assets_dir = Path(self.tmp.name) / "assets"
        self.assets_dir.mkdir()
        self.manifest = Path(self.tmp.name) / "manifest.json"

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def test_new_asset_added(self) -> None:
        content = b"export const x = 1;"
        make_asset(self.assets_dir, "new-module.mjs", content)
        sz, sha = len(content), sha256_hex(content)
        make_manifest(self.assets_dir, self.manifest, [])

        result = run_tool("--write", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")

        updated = json.loads(self.manifest.read_text(encoding="utf-8"))
        self.assertEqual(len(updated["assets"]), 1)
        e = updated["assets"][0]
        self.assertEqual(e["name"], "new-module.mjs")
        self.assertEqual(e["size"], sz)
        self.assertEqual(e["sha256"], sha)
        self.assertEqual(e["moduleType"], "esm")

    def test_deleted_asset_removed(self) -> None:
        make_manifest(
            self.assets_dir,
            self.manifest,
            [{"name": "gone.mjs", "path": "gone.mjs", "role": "loader-support",
              "moduleType": "esm", "size": 10, "sha256": "c" * 64}],
        )
        result = run_tool("--write", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")
        updated = json.loads(self.manifest.read_text(encoding="utf-8"))
        self.assertEqual(len(updated["assets"]), 0)


class ContentChangedTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.assets_dir = Path(self.tmp.name) / "assets"
        self.assets_dir.mkdir()
        self.manifest = Path(self.tmp.name) / "manifest.json"

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def test_changed_content_updates_digest_and_size(self) -> None:
        content1 = b"a"
        content2 = b"ab"
        make_asset(self.assets_dir, "mod.mjs", content2)
        sz1, sha1 = len(content1), sha256_hex(content1)
        make_manifest(
            self.assets_dir,
            self.manifest,
            [{"name": "mod.mjs", "path": "mod.mjs", "role": "loader-support",
              "moduleType": "esm", "size": sz1, "sha256": sha1}],
        )
        result = run_tool("--write", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")

        updated = json.loads(self.manifest.read_text(encoding="utf-8"))
        e = updated["assets"][0]
        self.assertEqual(e["size"], len(content2))
        self.assertEqual(e["sha256"], sha256_hex(content2))

    def test_binary_asset_hashing(self) -> None:
        content = bytes(range(256))
        make_asset(self.assets_dir, "data.cjs", content)
        make_manifest(self.assets_dir, self.manifest, [])

        result = run_tool("--write", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")

        updated = json.loads(self.manifest.read_text(encoding="utf-8"))
        e = updated["assets"][0]
        self.assertEqual(e["size"], 256)
        self.assertEqual(e["sha256"], sha256_hex(content))

    def test_empty_file(self) -> None:
        make_asset(self.assets_dir, "empty.mjs", b"")
        make_manifest(self.assets_dir, self.manifest, [])

        result = run_tool("--write", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")

        updated = json.loads(self.manifest.read_text(encoding="utf-8"))
        e = updated["assets"][0]
        self.assertEqual(e["size"], 0)
        self.assertEqual(e["sha256"], sha256_hex(b""))


class NestedAssetTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.assets_dir = Path(self.tmp.name) / "assets"
        self.assets_dir.mkdir()
        self.manifest = Path(self.tmp.name) / "manifest.json"

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def test_nested_asset_path(self) -> None:
        sub = self.assets_dir / "sub"
        sub.mkdir()
        content = b"nested"
        p = sub / "nested.mjs"
        p.write_bytes(content)
        make_manifest(self.assets_dir, self.manifest, [])

        result = run_tool("--write", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")

        updated = json.loads(self.manifest.read_text(encoding="utf-8"))
        self.assertEqual(len(updated["assets"]), 1)
        e = updated["assets"][0]
        self.assertEqual(e["path"], "sub/nested.mjs")
        self.assertEqual(e["size"], len(content))
        self.assertEqual(e["sha256"], sha256_hex(content))


class UnicodeSpaceTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.assets_dir = Path(self.tmp.name) / "assets"
        self.assets_dir.mkdir()
        self.manifest = Path(self.tmp.name) / "manifest.json"

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def test_unicode_filename(self) -> None:
        content = b"// unicode"
        make_asset(self.assets_dir, "émoji.mjs", content)
        make_manifest(self.assets_dir, self.manifest, [])

        result = run_tool("--write", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")

        updated = json.loads(self.manifest.read_text(encoding="utf-8"))
        self.assertEqual(len(updated["assets"]), 1)
        self.assertEqual(updated["assets"][0]["name"], "émoji.mjs")

    def test_space_in_filename(self) -> None:
        content = b"// space"
        make_asset(self.assets_dir, "my file.mjs", content)
        make_manifest(self.assets_dir, self.manifest, [])

        result = run_tool("--write", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")

        updated = json.loads(self.manifest.read_text(encoding="utf-8"))
        self.assertEqual(len(updated["assets"]), 1)
        self.assertEqual(updated["assets"][0]["path"], "my file.mjs")


class DeterministicTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.assets_dir = Path(self.tmp.name) / "assets"
        self.assets_dir.mkdir()
        self.manifest = Path(self.tmp.name) / "manifest.json"

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def test_deterministic_ordering(self) -> None:
        # Create files in non-alphabetical order.
        for name in ("z.mjs", "a.mjs", "m.mjs"):
            make_asset(self.assets_dir, name, b"x")
        make_manifest(self.assets_dir, self.manifest, [])

        run_tool("--write", "--manifest", str(self.manifest),
                 "--assets-dir", str(self.assets_dir))
        text1 = self.manifest.read_text(encoding="utf-8")

        # Delete and regenerate.
        self.manifest.unlink()
        make_manifest(self.assets_dir, self.manifest, [])
        run_tool("--write", "--manifest", str(self.manifest),
                 "--assets-dir", str(self.assets_dir))
        text2 = self.manifest.read_text(encoding="utf-8")

        self.assertEqual(text1, text2)
        updated = json.loads(text1)
        names = [e["name"] for e in updated["assets"]]
        self.assertEqual(names, sorted(names))

    def test_deterministic_json_formatting(self) -> None:
        content = b"x"
        make_asset(self.assets_dir, "test.mjs", content)
        make_manifest(self.assets_dir, self.manifest)

        run_tool("--write", "--manifest", str(self.manifest),
                 "--assets-dir", str(self.assets_dir))
        text1 = self.manifest.read_text(encoding="utf-8")

        self.manifest.unlink()
        make_manifest(self.assets_dir, self.manifest)
        run_tool("--write", "--manifest", str(self.manifest),
                 "--assets-dir", str(self.assets_dir))
        text2 = self.manifest.read_text(encoding="utf-8")

        self.assertEqual(text1, text2)
        self.assertTrue(text1.endswith("\n"))


class ValidationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.assets_dir = Path(self.tmp.name) / "assets"
        self.assets_dir.mkdir()
        self.manifest = Path(self.tmp.name) / "manifest.json"

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def test_malformed_manifest(self) -> None:
        self.manifest.write_text("not json", encoding="utf-8")
        result = run_tool("--check", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("malformed", result.stderr.lower())

    def test_duplicate_name(self) -> None:
        make_manifest(
            self.assets_dir,
            self.manifest,
            [
                {"name": "dup.mjs", "path": "a.mjs", "role": "loader-support",
                 "moduleType": "esm", "size": 10, "sha256": "a" * 64},
                {"name": "dup.mjs", "path": "b.mjs", "role": "loader-support",
                 "moduleType": "esm", "size": 20, "sha256": "b" * 64},
            ],
        )
        result = run_tool("--check", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("duplicate", result.stderr.lower())

    def test_duplicate_path(self) -> None:
        make_manifest(
            self.assets_dir,
            self.manifest,
            [
                {"name": "a.mjs", "path": "same.mjs", "role": "loader-support",
                 "moduleType": "esm", "size": 10, "sha256": "a" * 64},
                {"name": "b.mjs", "path": "same.mjs", "role": "loader-support",
                 "moduleType": "esm", "size": 20, "sha256": "b" * 64},
            ],
        )
        result = run_tool("--check", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("duplicate", result.stderr.lower())

    def test_path_traversal_rejected(self) -> None:
        make_manifest(
            self.assets_dir,
            self.manifest,
            [{"name": "bad", "path": "../etc/passwd", "role": "loader-support",
              "moduleType": "esm", "size": 10, "sha256": "a" * 64}],
        )
        result = run_tool("--check", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertNotEqual(result.returncode, 0)

    def test_invalid_module_type(self) -> None:
        make_manifest(
            self.assets_dir,
            self.manifest,
            [{"name": "bad.mjs", "path": "bad.mjs", "role": "loader-support",
              "moduleType": "wasm", "size": 10, "sha256": "a" * 64}],
        )
        result = run_tool("--check", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertNotEqual(result.returncode, 0)

    def test_invalid_sha256_format(self) -> None:
        make_manifest(
            self.assets_dir,
            self.manifest,
            [{"name": "bad.mjs", "path": "bad.mjs", "role": "loader-support",
              "moduleType": "esm", "size": 10, "sha256": "not-hex"}],
        )
        result = run_tool("--check", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertNotEqual(result.returncode, 0)

    def test_case_collision_detection(self) -> None:
        content = b"x"
        make_asset(self.assets_dir, "File.mjs", content)
        make_asset(self.assets_dir, "file.mjs", content)
        make_manifest(self.assets_dir, self.manifest, [])
        result = run_tool("--write", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("collision", result.stderr.lower())


class SymlinkTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.assets_dir = Path(self.tmp.name) / "assets"
        self.assets_dir.mkdir()
        self.manifest = Path(self.tmp.name) / "manifest.json"

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def test_symlinks_skipped(self) -> None:
        target = self.assets_dir / "real.mjs"
        target.write_bytes(b"real")
        link = self.assets_dir / "link.mjs"
        os.symlink(str(target), str(link))
        make_manifest(self.assets_dir, self.manifest, [])

        result = run_tool("--write", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")

        updated = json.loads(self.manifest.read_text(encoding="utf-8"))
        paths = [e["path"] for e in updated["assets"]]
        self.assertIn("real.mjs", paths)
        self.assertNotIn("link.mjs", paths)


class AtomicWriteTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.assets_dir = Path(self.tmp.name) / "assets"
        self.assets_dir.mkdir()
        self.manifest = Path(self.tmp.name) / "manifest.json"

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def test_scan_failure_preserves_manifest(self) -> None:
        # Write a valid manifest pointing to a file that exists.
        content = b"ok"
        make_asset(self.assets_dir, "test.mjs", content)
        sz, sha = len(content), sha256_hex(content)
        original = json.dumps({
            "schemaVersion": 2,
            "bundleVersion": "4",
            "assets": [{"name": "test.mjs", "path": "test.mjs",
                        "role": "loader-support", "moduleType": "esm",
                        "size": sz, "sha256": sha}],
        }, indent=2) + "\n"
        self.manifest.write_text(original, encoding="utf-8")

        # Delete the assets dir to cause scan failure.
        import shutil
        shutil.rmtree(str(self.assets_dir))

        result = run_tool("--write", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.manifest.read_text(encoding="utf-8"), original)

    def test_no_temp_files_left_after_success(self) -> None:
        make_asset(self.assets_dir, "test.mjs", b"x")
        make_manifest(self.assets_dir, self.manifest, [])
        before = set(os.listdir(str(self.manifest.parent)))
        # Remove the temp dir's own entries that aren't the manifest.
        before.discard("manifest.json")
        before.discard("assets")

        result = run_tool("--write", "--manifest", str(self.manifest),
                          "--assets-dir", str(self.assets_dir))
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")

        after = set(os.listdir(str(self.manifest.parent)))
        after.discard("manifest.json")
        after.discard("assets")
        self.assertEqual(before, after,
                         f"Temp files left: {after - before}")


class WrapperTests(unittest.TestCase):
    def test_sh_wrapper_exists(self) -> None:
        wrapper = ROOT / "tools" / "update-runtime-assets.sh"
        self.assertTrue(wrapper.is_file(), f"Missing {wrapper}")

    def test_ps1_wrapper_exists(self) -> None:
        wrapper = ROOT / "tools" / "update-runtime-assets.ps1"
        self.assertTrue(wrapper.is_file(), f"Missing {wrapper}")

    def test_sh_wrapper_forwards_args(self) -> None:
        wrapper = ROOT / "tools" / "update-runtime-assets.sh"
        result = subprocess.run(
            ["sh", str(wrapper), "--help"],
            cwd=ROOT,
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")
        self.assertIn("usage", result.stdout.lower())


class RepositoryCheckTests(unittest.TestCase):
    def test_checked_in_manifest_matches_assets(self) -> None:
        """Repository-level check: the committed manifest matches current assets."""
        result = run_tool("--check")
        self.assertEqual(
            result.returncode, 0,
            f"Checked-in manifest is stale. Run --write.\n{result.stderr}",
        )


if __name__ == "__main__":
    unittest.main()
