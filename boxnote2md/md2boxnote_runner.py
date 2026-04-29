"""md → .boxnote 変換ランナー。"""

from __future__ import annotations

import json
import sys
from dataclasses import dataclass, field
from pathlib import Path

from boxnote2md.md import parse_markdown, wrap_envelope


@dataclass
class Md2BoxOptions:
    input_path: Path
    out_dir: Path
    recursive: bool = True
    flat: bool = False
    overwrite: bool = False
    dry_run: bool = False
    verbose: bool = False


@dataclass
class Md2BoxSummary:
    success: int = 0
    skipped: int = 0
    failed: int = 0
    failures: list[tuple[Path, str]] = field(default_factory=list)


def _log(msg: str) -> None:
    print(msg, file=sys.stderr)


def _collect(opts: Md2BoxOptions) -> tuple[list[Path], Path]:
    inp = opts.input_path
    if inp.is_file():
        return [inp], inp
    if inp.is_dir():
        files = sorted(inp.rglob("*.md") if opts.recursive else inp.glob("*.md"))
        return files, inp
    raise FileNotFoundError(f"input not found or unsupported: {inp}")


def _derive_output(src: Path, *, input_root: Path, out_dir: Path, flat: bool) -> Path:
    if flat or src == input_root:
        return out_dir / f"{src.stem}.boxnote"
    rel = src.relative_to(input_root).with_suffix(".boxnote")
    return out_dir / rel


def _resolve_collision(path: Path, taken: set[Path]) -> Path:
    if path not in taken:
        return path
    base = path.with_suffix("")
    i = 1
    while True:
        cand = Path(f"{base}-{i}.boxnote")
        if cand not in taken:
            return cand
        i += 1


def _convert_one(src: Path, dest: Path, opts: Md2BoxOptions) -> str:
    md_text = src.read_text(encoding="utf-8")
    doc = parse_markdown(md_text)
    envelope = wrap_envelope(doc)
    if dest.exists() and not opts.overwrite:
        return "skipped"
    if opts.dry_run:
        return "would-write"
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_text(
        json.dumps(envelope, ensure_ascii=False, separators=(",", ":")), encoding="utf-8"
    )
    return "written"


def run(opts: Md2BoxOptions) -> Md2BoxSummary:
    summary = Md2BoxSummary()

    try:
        files, input_root = _collect(opts)
    except FileNotFoundError as e:
        _log(f"error: {e}")
        summary.failed = 1
        summary.failures.append((opts.input_path, str(e)))
        return summary

    if not files:
        _log(f"no .md files found under {opts.input_path}")
        return summary

    if opts.verbose:
        _log(f"found {len(files)} file(s)")

    taken: set[Path] = set()
    for src in files:
        dest = _derive_output(
            src, input_root=input_root, out_dir=opts.out_dir, flat=opts.flat
        )
        dest = _resolve_collision(dest, taken)
        taken.add(dest)
        if opts.verbose:
            _log(f"converting {src}")
        try:
            status = _convert_one(src, dest, opts)
        except Exception as e:  # noqa: BLE001
            summary.failed += 1
            summary.failures.append((src, repr(e)))
            _log(f"FAILED: {src}: {e}")
            continue

        if opts.verbose or opts.dry_run:
            _log(f"  -> {dest} [{status}]")
        if status == "skipped":
            summary.skipped += 1
        else:
            summary.success += 1

    print(
        f"summary: success={summary.success} skipped={summary.skipped} failed={summary.failed}",
        file=sys.stderr,
    )
    if summary.failures:
        print("failures:", file=sys.stderr)
        for path, msg in summary.failures:
            print(f"  - {path}: {msg}", file=sys.stderr)

    return summary
