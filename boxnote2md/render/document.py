from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

Node = dict[str, Any]


@dataclass
class RenderContext:
    """変換中の状態を保持する。"""

    keep_styles: bool = False
    image_mode: str = "download"
    # 画像保存先 (download モード時)。MDからの相対参照解決にも使う
    image_dir: Path | None = None
    # 出力 .md ファイル自身のパス (画像相対参照を計算するため)
    md_path: Path | None = None
    # 画像取得結果のキャッシュ・記録 (Phase 5 で利用)
    image_results: list[dict[str, Any]] = field(default_factory=list)
    # 警告ログ
    warnings: list[str] = field(default_factory=list)
    # ネスト中のリスト深さ (リストレンダラが管理)
    list_depth: int = 0
    # テーブルセル内かどうか (改行抑止用)
    in_table_cell: bool = False

    def warn(self, msg: str) -> None:
        self.warnings.append(msg)


# 各ノードタイプのレンダラを登録するディスパッチ表。
# 遅延 import 回避のため document.py 末尾で blocks/inline からまとめて埋める。
BLOCK_RENDERERS: dict[str, Callable[[Node, RenderContext], str]] = {}
INLINE_RENDERERS: dict[str, Callable[[Node, RenderContext], str]] = {}


def render_document(doc: Node, ctx: RenderContext) -> str:
    """`doc` ノード → Markdown 文字列。"""
    if doc.get("type") != "doc":
        raise ValueError(f"expected doc node, got: {doc.get('type')!r}")
    parts: list[str] = []
    for child in doc.get("content", []) or []:
        rendered = render_block(child, ctx)
        if rendered:
            parts.append(rendered)
    md = "\n\n".join(p.rstrip() for p in parts if p.strip())
    return md.rstrip() + "\n"


def render_block(node: Node, ctx: RenderContext) -> str:
    t = node.get("type")
    renderer = BLOCK_RENDERERS.get(t)
    if renderer is None:
        ctx.warn(f"unknown block type: {t!r} — passing through children")
        return render_children_inline(node, ctx)
    return renderer(node, ctx)


def render_children_inline(node: Node, ctx: RenderContext) -> str:
    parts: list[str] = []
    for child in node.get("content", []) or []:
        if child.get("type") == "text":
            parts.append(render_text(child, ctx))
        elif child.get("type") in INLINE_RENDERERS:
            parts.append(INLINE_RENDERERS[child["type"]](child, ctx))
        elif child.get("type") in BLOCK_RENDERERS:
            # block が inline 文脈に出ても素直に展開
            parts.append(BLOCK_RENDERERS[child["type"]](child, ctx))
        else:
            ctx.warn(f"unknown inline child type: {child.get('type')!r}")
    return "".join(parts)


def render_text(node: Node, ctx: RenderContext) -> str:
    # 循環 import を避けるため遅延 import
    from boxnote2md.render.inline import render_text_with_marks

    return render_text_with_marks(node, ctx)


def _register_renderers() -> None:
    """ディスパッチ表に各レンダラを登録する。"""
    from boxnote2md.render import blocks, images, inline, lists, tables

    BLOCK_RENDERERS.update(blocks.RENDERERS)
    BLOCK_RENDERERS.update(lists.RENDERERS)
    BLOCK_RENDERERS.update(tables.RENDERERS)
    BLOCK_RENDERERS.update(images.RENDERERS)
    INLINE_RENDERERS.update(inline.RENDERERS)
    INLINE_RENDERERS.update(images.RENDERERS)  # image はインライン文脈にも出る可能性


_register_renderers()
