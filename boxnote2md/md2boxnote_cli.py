from __future__ import annotations

import argparse
import sys
from pathlib import Path

from boxnote2md import __version__
from boxnote2md.md2boxnote_runner import Md2BoxOptions, run


def _build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        prog="md2boxnote",
        description="Markdown (.md) を Box Note (.boxnote) に変換する CLI。",
    )
    p.add_argument("--version", action="version", version=f"%(prog)s {__version__}")
    p.add_argument(
        "input",
        metavar="<input>",
        type=Path,
        help=".md ファイル または ディレクトリ",
    )
    p.add_argument(
        "-o",
        "--out",
        dest="out_dir",
        type=Path,
        default=Path("./out"),
        help="出力先ディレクトリ (default: ./out)",
    )
    p.add_argument(
        "-r",
        "--recursive",
        dest="recursive",
        action=argparse.BooleanOptionalAction,
        default=True,
        help="ディレクトリ入力時に再帰探索する (default: 有効)",
    )
    p.add_argument("--flat", action="store_true", help="出力先直下にフラット配置する")
    p.add_argument("--overwrite", action="store_true", help="既存 .boxnote を上書きする")
    p.add_argument("--dry-run", action="store_true", help="書き込みせず処理予定だけ表示する")
    p.add_argument("-v", "--verbose", action="store_true", help="詳細ログを出力する")
    return p


def main(argv: list[str] | None = None) -> int:
    parser = _build_parser()
    args = parser.parse_args(argv)

    if not args.input.exists():
        parser.error(f"input not found: {args.input}")

    opts = Md2BoxOptions(
        input_path=args.input,
        out_dir=args.out_dir,
        recursive=args.recursive,
        flat=args.flat,
        overwrite=args.overwrite,
        dry_run=args.dry_run,
        verbose=args.verbose,
    )
    summary = run(opts)
    return 0 if summary.failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
