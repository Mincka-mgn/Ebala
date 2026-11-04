#!/usr/bin/env python3
"""Prepare Ebala beta archives from uploaded Windows ZIP and tracked repository files."""
from __future__ import annotations

import hashlib
import os
from pathlib import Path, PurePosixPath
import shutil
import stat
import struct
import subprocess
import sys
import tempfile
import zipfile

ROOT = Path.cwd()
VERSION = sys.argv[1] if len(sys.argv) > 1 else "0.55"
INPUT = ROOT / f"Ebala_{VERSION}.zip"
OUT = ROOT / "dist"
DIRECTORIES = ("Maps", "Catalog", "Databases", "Resources")
NOTES = (f"CHANGES_{VERSION}.txt", f"TESTPLAN_WINDOWS_{VERSION}.txt", "README.txt")
MAX_FILES = 20000
MAX_TOTAL = 2_000_000_000


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def entry_path(name: str) -> PurePosixPath:
    s = name.replace("\\", "/")
    p = PurePosixPath(s)
    if not s or p.is_absolute() or ".." in p.parts or ":" in s:
        raise ValueError(f"Unsafe ZIP member: {name!r}")
    if any(x in (".git", "__MACOSX") for x in p.parts):
        raise ValueError(f"Disallowed ZIP member: {name!r}")
    return p


def verify_pe_gui(exe: Path) -> None:
    with exe.open("rb") as f:
        head = f.read(256)
        if head[:2] != b"MZ" or len(head) < 64:
            raise ValueError("Windows executable is not an MZ/PE file")
        pe_off = struct.unpack_from("<I", head, 0x3c)[0]
        f.seek(pe_off)
        header = f.read(24 + 72)
    if len(header) < 96 or header[:4] != b"PE\x00\x00":
        raise ValueError("Invalid PE header")
    machine = struct.unpack_from("<H", header, 4)[0]
    magic = struct.unpack_from("<H", header, 24)[0]
    subsystem = struct.unpack_from("<H", header, 24 + 68)[0]
    if (machine, magic, subsystem) != (0x8664, 0x20b, 2):
        raise ValueError(f"Expected Windows x64 GUI (0x8664, PE32+, Subsystem 2); got {machine:#x}, {magic:#x}, {subsystem}")
    print("PASS: Windows x64 GUI / PE Subsystem=2")


def write_zip(target: Path, basedir: Path, rels: list[str]) -> None:
    target.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(target, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=6, allowZip64=True) as z:
        for rel in rels:
            p = basedir / rel
            if not p.is_file():
                raise ValueError(f"Source file missing while packing: {rel}")
            z.write(p, rel)
    with zipfile.ZipFile(target, "r") as z:
        bad = z.testzip()
        if bad:
            raise ValueError(f"Output archive failed CRC check: {bad}")
        if sorted(z.namelist()) != sorted(rels):
            raise ValueError("Output archive file list differs from input")
    print(f"PASS: {target.name}: {len(rels)} files, {target.stat().st_size:,} bytes, sha256={sha256(target)}")


def prepare_windows() -> Path:
    if not INPUT.is_file():
        raise ValueError(f"Uploaded Windows archive missing: {INPUT.name}")
    with tempfile.TemporaryDirectory() as td:
        stage = Path(td) / "package"
        stage.mkdir()
        with zipfile.ZipFile(INPUT, "r") as z:
            if z.testzip():
                raise ValueError("Uploaded Windows ZIP has CRC errors")
            entries = [(info, entry_path(info.filename)) for info in z.infolist() if not info.is_dir()]
            print(f"Windows ZIP: {len(entries)} members, first 35: {[str(p) for _, p in entries[:35]]}", flush=True)
            if not entries or len(entries) > MAX_FILES or sum(i.file_size for i, _ in entries) > MAX_TOTAL:
                raise ValueError("Windows archive is empty or exceeds safety limits")
            exes = [p for _, p in entries if p.name.lower() == f"ebala_{VERSION}.exe"]
            if len(exes) != 1:
                raise ValueError(f"Expected exactly one Ebala_{VERSION}.exe, found {exes}")
            parent = exes[0].parent
            if parent.parts and not all(p.parts[:len(parent.parts)] == parent.parts for _, p in entries):
                raise ValueError(f"Cannot safely flatten Windows ZIP: EXE in {parent}, other entries outside")
            seen = set()
            for info, path in entries:
                mode = (info.external_attr >> 16) & 0xffff
                if stat.S_ISLNK(mode):
                    raise ValueError(f"ZIP symlink rejected: {path}")
                rel = PurePosixPath(*path.parts[len(parent.parts):]) if parent.parts else path
                if not rel.parts:
                    raise ValueError(f"Empty normalized path: {path}")
                key = str(rel).casefold()
                if key in seen:
                    raise ValueError(f"Duplicate normalized Windows path: {rel}")
                seen.add(key)
                target = stage.joinpath(*rel.parts)
                target.parent.mkdir(parents=True, exist_ok=True)
                with z.open(info) as src, target.open("wb") as dst:
                    shutil.copyfileobj(src, dst, 1024 * 1024)

        exe = stage / f"Ebala_{VERSION}.exe"
        verify_pe_gui(exe)
        added = {}
        for folder in DIRECTORIES:
            source_dir = ROOT / folder
            if not source_dir.is_dir():
                raise ValueError(f"Required resource directory missing: {folder}")
            count = 0
            for source in source_dir.rglob("*"):
                if not source.is_file():
                    continue
                dest = stage / source.relative_to(ROOT)
                if not dest.exists():
                    dest.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(source, dest)
                    count += 1
            added[folder] = count
        print(f"Added missing resources from repository: {added}", flush=True)
        for note in NOTES:
            if not (stage / note).exists() and (ROOT / note).is_file():
                shutil.copy2(ROOT / note, stage / note)
        maps = [x for x in (stage / "Maps").rglob("*") if x.is_file()]
        if len(maps) < 30:
            raise ValueError(f"Too few map files in Windows package: {len(maps)}")
        rels = sorted(p.relative_to(stage).as_posix() for p in stage.rglob("*") if p.is_file())
        if f"Ebala_{VERSION}.exe" not in rels:
            raise ValueError("Windows EXE missing from archive root")
        for folder in DIRECTORIES:
            if not any(p.startswith(folder + "/") for p in rels):
                raise ValueError(f"Missing packaged directory: {folder}")
        out = OUT / f"Ebala_{VERSION}.zip"
        write_zip(out, stage, rels)
        print(f"PASS: Windows package includes {len(maps)} Maps files and all 4 resource directories", flush=True)
        return out


def prepare_sources() -> Path:
    tracked = subprocess.check_output(["git", "ls-files", "-z"], cwd=ROOT).split(b"\x00")
    paths = []
    for raw in tracked:
        if not raw:
            continue
        rel = os.fsdecode(raw).replace("\\", "/")
        if rel == INPUT.name or rel.startswith(".github/"):
            continue
        if (ROOT / rel).is_file():
            paths.append(rel)
    for must in ("PROJECT.md", "main_windows.go", "model.go", f"CHANGES_{VERSION}.txt"):
        if must not in paths:
            raise ValueError(f"Required source file missing: {must}")
    if not any(x.startswith("Maps/") for x in paths):
        raise ValueError("Source archive would lack Maps")
    out = OUT / f"Ebala_{VERSION}_Sources.zip"
    write_zip(out, ROOT, sorted(paths))
    return out


if __name__ == "__main__":
    OUT.mkdir(exist_ok=True)
    prepare_windows()
    prepare_sources()
    print(f"READY: Ebala {VERSION} beta package archives prepared; no release published")
