#!/usr/bin/env python3
"""Package Ebala 0.64 beta from the exact uploaded Windows and source archives.

The source ZIP is recovered by the workflow from the upload commit because
the regular source-import workflow removes the uploaded ZIP after extracting.
This script DOES NOT compile or change the application.
"""
from __future__ import annotations

import importlib.util
from pathlib import Path, PurePosixPath
import shutil
import stat
import tempfile
import zipfile

ROOT = Path.cwd()
BASE = ROOT / ".github/scripts/package_ebala_beta.py"
SOURCE = ROOT / "_input_064_sources.zip"
VERSION = "0.64"

spec = importlib.util.spec_from_file_location("ebala_packager", BASE)
if spec is None or spec.loader is None:
    raise SystemExit("Cannot load existing package validator")
pkg = importlib.util.module_from_spec(spec)
spec.loader.exec_module(pkg)
pkg.VERSION = VERSION
pkg.INPUT = ROOT / "Ebala_0.64_Windows.zip"

def source_files():
    if not SOURCE.is_file():
        raise ValueError("Original uploaded Ebala_0.64_Sources.zip was not recovered from history")
    with tempfile.TemporaryDirectory() as tmp:
        stage = Path(tmp)
        with zipfile.ZipFile(SOURCE) as z:
            bad = z.testzip()
            if bad:
                raise ValueError(f"Source ZIP CRC failure: {bad}")
            entries = [(i, pkg.entry_path(i.filename)) for i in z.infolist() if not i.is_dir()]
            if len(entries) < 100 or len(entries) > 20000:
                raise ValueError(f"Unexpected source ZIP file count: {len(entries)}")
            paths = {str(p) for _, p in entries}
            if not {"PROJECT.md", "main_windows.go", "model.go", "CHANGES_0.64.txt"}.issubset(paths):
                raise ValueError("Original 0.64 sources are incomplete or have an unexpected parent directory")
            if sum(p.endswith(".go") for p in paths) < 150:
                raise ValueError("Unexpectedly few Go source files")
            if b'const ebalaVersion = "0.64"' not in z.read("main_windows.go"):
                raise ValueError("Source version is not 0.64")
            seen = set()
            for info, rel in entries:
                mode = (info.external_attr >> 16) & 0xffff
                if stat.S_ISLNK(mode):
                    raise ValueError(f"Symlink in uploaded source ZIP: {rel}")
                casekey = str(rel).casefold()
                if casekey in seen:
                    raise ValueError(f"Duplicate source path: {rel}")
                seen.add(casekey)
                dest = stage.joinpath(*rel.parts)
                dest.parent.mkdir(parents=True, exist_ok=True)
                with z.open(info) as src, dest.open("wb") as target:
                    shutil.copyfileobj(src, target, 1024 * 1024)

        # Maps may be intentionally stored outside the original source ZIP.
        # Preserve original source files unchanged and only add missing maps.
        added = 0
        for src in (ROOT / "Maps").rglob("*"):
            if not src.is_file():
                continue
            dest = stage / src.relative_to(ROOT)
            if not dest.exists():
                dest.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(src, dest)
                added += 1
        maps = [p for p in (stage / "Maps").rglob("*") if p.is_file()]
        if len(maps) < 30:
            raise ValueError(f"Source archive is missing maps: {len(maps)}")
        assert (stage / "PROJECT.md").read_bytes() != (ROOT / "PROJECT.md").read_bytes(), (
            "Expected newer PROJECT.md in uploaded 0.64 sources; check source-import logic"
        )
        rels = sorted(p.relative_to(stage).as_posix() for p in stage.rglob("*") if p.is_file())
        out = ROOT / "dist/Ebala_0.64_Sources.zip"
        pkg.write_zip(out, stage, rels)
        print(f"PASS: full original 0.64 sources + {added} maps; "
              f"{len(maps)} map files; PROJECT.md from original uploaded 0.64 source", flush=True)


if __name__ == "__main__":
    windows = pkg.prepare_windows()
    target = ROOT / "dist/Ebala_0.64_Windows.zip"
    windows.replace(target)
    with zipfile.ZipFile(target) as z:
        if z.testzip():
            raise ValueError("Windows ZIP failed final CRC check")
        if "Ebala_0.64.exe" not in z.namelist():
            raise ValueError("Executable not at top level")
    print(f"PASS: final Windows archive {target.name}, sha256={pkg.sha256(target)}", flush=True)
    source_files()
    print("READY: both Ebala 0.64 packages; NO program source modified", flush=True)
