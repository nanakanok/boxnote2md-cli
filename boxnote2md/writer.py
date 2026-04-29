"""出力先パス解決と .md 書き出し。"""

from __future__ import annotations

from pathlib import Path


def derive_output_path(
    src: Path,
    *,
    input_root: Path,
    out_dir: Path,
    flat: bool,
) -> Path:
    """入力ファイル `src` に対する出力 .md パスを決める。

    - flat=False: out_dir / (input_root からの相対パス).md
    - flat=True : out_dir / src.stem.md (衝突は呼び出し側で連番付与)
    - 入力がファイル単体 (input_root == src) のときは out_dir / src.stem.md
    """
    if flat or src == input_root:
        return out_dir / f"{src.stem}.md"
    rel = src.relative_to(input_root).with_suffix(".md")
    return out_dir / rel


def resolve_collision(path: Path, taken: set[Path]) -> Path:
    """同一出力パスが既に取られていれば連番を振る (--flat 時用)。"""
    if path not in taken:
        return path
    base = path.with_suffix("")
    suffix = path.suffix
    i = 1
    while True:
        candidate = Path(f"{base}-{i}{suffix}")
        if candidate not in taken:
            return candidate
        i += 1


def write_markdown(path: Path, content: str, *, overwrite: bool, dry_run: bool) -> str:
    """書き出して結果ステータスを返す: "written" / "skipped" / "would-write"。"""
    if path.exists() and not overwrite:
        return "skipped"
    if dry_run:
        return "would-write"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")
    return "written"
