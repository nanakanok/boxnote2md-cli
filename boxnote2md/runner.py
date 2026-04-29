"""ファイル/ディレクトリ単位の変換オーケストレータ。"""

from __future__ import annotations

import sys
from dataclasses import dataclass, field
from pathlib import Path

from boxnote2md.reader import BoxNoteParseError, read_boxnote
from boxnote2md.render import RenderContext, render_document
from boxnote2md.writer import derive_output_path, resolve_collision, write_markdown


@dataclass
class RunOptions:
    input_path: Path
    out_dir: Path
    recursive: bool = True
    flat: bool = False
    overwrite: bool = False
    image_mode: str = "download"
    image_dir: Path | None = None
    keep_styles: bool = False
    dry_run: bool = False
    verbose: bool = False


@dataclass
class RunSummary:
    success: int = 0
    skipped: int = 0
    failed: int = 0
    failures: list[tuple[Path, str]] = field(default_factory=list)


def _log(msg: str, *, verbose: bool = False, ctx_verbose: bool = False) -> None:
    if verbose and not ctx_verbose:
        return
    print(msg, file=sys.stderr)


def _collect_inputs(opts: RunOptions) -> tuple[list[Path], Path]:
    """変換対象ファイルのリストと、出力パス計算用のルートを返す。"""
    inp = opts.input_path
    if inp.is_file():
        return [inp], inp
    if inp.is_dir():
        files = sorted(inp.rglob("*.boxnote") if opts.recursive else inp.glob("*.boxnote"))
        return files, inp
    raise FileNotFoundError(f"input not found or unsupported: {inp}")


def _convert_one(src: Path, dest: Path, opts: RunOptions) -> None:
    doc = read_boxnote(src)
    ctx = RenderContext(
        keep_styles=opts.keep_styles,
        image_mode=opts.image_mode,
        image_dir=opts.image_dir,
        md_path=dest,
    )
    md = render_document(doc, ctx)
    if opts.verbose:
        for w in ctx.warnings:
            _log(f"  warn: {w}", verbose=True, ctx_verbose=True)
    status = write_markdown(dest, md, overwrite=opts.overwrite, dry_run=opts.dry_run)
    if opts.verbose or opts.dry_run:
        _log(f"  -> {dest} [{status}]", verbose=True, ctx_verbose=True)


def run(opts: RunOptions) -> RunSummary:
    summary = RunSummary()

    try:
        files, input_root = _collect_inputs(opts)
    except FileNotFoundError as e:
        _log(f"error: {e}", verbose=True, ctx_verbose=True)
        summary.failed = 1
        summary.failures.append((opts.input_path, str(e)))
        return summary

    if not files:
        _log(
            f"no .boxnote files found under {opts.input_path}",
            verbose=True,
            ctx_verbose=True,
        )
        return summary

    if opts.verbose:
        _log(f"found {len(files)} file(s)", verbose=True, ctx_verbose=True)

    taken: set[Path] = set()
    for src in files:
        dest = derive_output_path(
            src,
            input_root=input_root,
            out_dir=opts.out_dir,
            flat=opts.flat,
        )
        dest = resolve_collision(dest, taken)
        taken.add(dest)

        if opts.verbose:
            _log(f"converting {src}", verbose=True, ctx_verbose=True)

        try:
            existed_before = dest.exists()
            _convert_one(src, dest, opts)
        except BoxNoteParseError as e:
            summary.failed += 1
            summary.failures.append((src, str(e)))
            _log(f"FAILED: {src}: {e}", verbose=True, ctx_verbose=True)
            continue
        except Exception as e:  # noqa: BLE001
            summary.failed += 1
            summary.failures.append((src, repr(e)))
            _log(f"FAILED: {src}: {e}", verbose=True, ctx_verbose=True)
            continue

        # skipped 判定: 既存があり overwrite=False
        if existed_before and not opts.overwrite:
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
