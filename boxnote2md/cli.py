from __future__ import annotations

import argparse
import sys
from pathlib import Path

from boxnote2md import __version__
from boxnote2md.runner import RunOptions, run


def _build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        prog="boxnote2md",
        description="Box Note (.boxnote) を Markdown に変換する CLI。",
    )
    p.add_argument("--version", action="version", version=f"%(prog)s {__version__}")
    p.add_argument(
        "input",
        metavar="<input>",
        type=Path,
        help=".boxnote ファイル または ディレクトリ",
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
    p.add_argument("--overwrite", action="store_true", help="既存 .md を上書きする")
    p.add_argument(
        "--image-mode",
        choices=("download", "url"),
        default="download",
        help="画像の扱い (default: download)",
    )
    p.add_argument(
        "--image-dir",
        type=Path,
        default=None,
        help="画像保存先 (default: <out>/images)",
    )
    p.add_argument(
        "--keep-styles",
        action="store_true",
        help="font_size/font_color/highlight/alignment を HTML として残す",
    )
    p.add_argument("--dry-run", action="store_true", help="書き込みせず処理予定だけ表示する")
    p.add_argument("-v", "--verbose", action="store_true", help="詳細ログを出力する")
    return p


def main(argv: list[str] | None = None) -> int:
    parser = _build_parser()
    args = parser.parse_args(argv)

    if not args.input.exists():
        parser.error(f"input not found: {args.input}")

    opts = RunOptions(
        input_path=args.input,
        out_dir=args.out_dir,
        recursive=args.recursive,
        flat=args.flat,
        overwrite=args.overwrite,
        image_mode=args.image_mode,
        image_dir=args.image_dir if args.image_dir else args.out_dir / "images",
        keep_styles=args.keep_styles,
        dry_run=args.dry_run,
        verbose=args.verbose,
    )
    summary = run(opts)
    return 0 if summary.failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
