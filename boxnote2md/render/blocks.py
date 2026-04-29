"""ブロックレベルノードの Markdown 化。"""

from __future__ import annotations

from boxnote2md.render.document import (
    BLOCK_RENDERERS,
    Node,
    RenderContext,
    render_block,
    render_children_inline,
)


def render_paragraph(node: Node, ctx: RenderContext) -> str:
    inner = render_children_inline(node, ctx)
    return inner  # 段落間の \n\n は document.py 側で結合


def render_heading(node: Node, ctx: RenderContext) -> str:
    level = int(node.get("attrs", {}).get("level", 1) or 1)
    level = max(1, min(level, 6))
    inner = render_children_inline(node, ctx).strip()
    return f"{'#' * level} {inner}"


def render_horizontal_rule(_node: Node, _ctx: RenderContext) -> str:
    return "---"


def _render_children_blocks(node: Node, ctx: RenderContext) -> list[str]:
    """ブロック子要素を順にレンダリングし、文字列リストで返す。"""
    parts: list[str] = []
    for child in node.get("content", []) or []:
        ct = child.get("type")
        if ct == "paragraph":
            parts.append(render_children_inline(child, ctx))
        elif ct in BLOCK_RENDERERS:
            parts.append(render_block(child, ctx))
        else:
            ctx.warn(f"unknown child of {node.get('type')!r}: {ct!r}")
            parts.append(render_children_inline(child, ctx))
    return [p for p in parts if p]


def render_blockquote(node: Node, ctx: RenderContext) -> str:
    blocks = _render_children_blocks(node, ctx)
    body = "\n\n".join(blocks)
    return "\n".join(f"> {ln}" if ln else ">" for ln in body.split("\n"))


def render_code_block(node: Node, ctx: RenderContext) -> str:
    language = (node.get("attrs", {}).get("language") or "").strip()
    # text ノードを連結 (marks は無視; コード内の装飾は MD では表現できない)
    text_parts: list[str] = []
    for child in node.get("content", []) or []:
        if child.get("type") == "text":
            text_parts.append(child.get("text", ""))
    code = "".join(text_parts)
    # 言語名は Markdown 慣習で小文字化
    fence_lang = language.lower() if language else ""
    return f"```{fence_lang}\n{code}\n```"


def render_call_out_box(node: Node, ctx: RenderContext) -> str:
    """call_out_box は MD に対応物が無いので blockquote に退避。

    emoji があれば先頭段落に付与。--keep-styles 時は HTML <div> に背景色付き。
    """
    attrs = node.get("attrs", {}) or {}
    emoji = attrs.get("emoji") or ""
    bg = attrs.get("backgroundColor")

    blocks = _render_children_blocks(node, ctx)
    if not blocks:
        return ""

    if emoji:
        blocks[0] = f"{emoji} {blocks[0]}"

    if ctx.keep_styles and bg:
        body = "\n\n".join(blocks)
        return f'<div style="background-color:{bg}">\n\n{body}\n\n</div>'

    body = "\n\n".join(blocks)
    return "\n".join(f"> {ln}" if ln else ">" for ln in body.split("\n"))


RENDERERS = {
    "paragraph": render_paragraph,
    "heading": render_heading,
    "horizontal_rule": render_horizontal_rule,
    "blockquote": render_blockquote,
    "code_block": render_code_block,
    "call_out_box": render_call_out_box,
}
