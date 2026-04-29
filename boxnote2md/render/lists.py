"""リスト系ノードの Markdown 化。

bullet_list / ordered_list / check_list を扱う。各 *_list の子は list_item
(check_list の場合は check_list_item)。list_item の中身は通常 paragraph 1個以上だが、
ネストリストを直接含むこともある。
"""

from __future__ import annotations

from boxnote2md.render.document import (
    BLOCK_RENDERERS,
    Node,
    RenderContext,
    render_block,
    render_children_inline,
)

INDENT = "    "  # GFM のリスト項目の継続行は4スペース


def _render_list_item_inner(item: Node, ctx: RenderContext) -> str:
    """list_item / check_list_item の中身を Markdown 段落列に変換。

    1段落目はマーカー直後にインライン展開、2段目以降は空行+インデントで継続。
    ネストリスト (bullet_list 等) は再帰的にレンダリングして、行ごとにインデント。
    """
    children = item.get("content", []) or []
    rendered_blocks: list[str] = []
    for child in children:
        ct = child.get("type")
        if ct == "paragraph":
            rendered_blocks.append(render_children_inline(child, ctx))
        elif ct in ("bullet_list", "ordered_list", "check_list"):
            rendered_blocks.append(render_block(child, ctx))
        else:
            # heading / blockquote / code_block 等もリスト内に出る可能性 → render_block 経由
            renderer = BLOCK_RENDERERS.get(ct)
            if renderer is not None:
                rendered_blocks.append(renderer(child, ctx))
            else:
                ctx.warn(f"unknown list_item child: {ct!r}")
                rendered_blocks.append(render_children_inline(child, ctx))
    return rendered_blocks


def _join_list_item(prefix: str, blocks: list[str]) -> str:
    """先頭ブロックを prefix と連結し、残りを 4 スペースインデントで続ける。

    ネストリストはすでに改行入りで返ってくるので、各行をインデント。
    """
    if not blocks:
        return prefix.rstrip()

    out_lines: list[str] = []
    first = blocks[0]
    # 先頭ブロックが複数行 (例: hard_break) のときも先頭行に prefix
    first_lines = first.split("\n")
    out_lines.append(f"{prefix}{first_lines[0]}")
    for ln in first_lines[1:]:
        out_lines.append(f"{INDENT}{ln}" if ln else "")

    for blk in blocks[1:]:
        out_lines.append("")  # 空行で段落区切り
        for ln in blk.split("\n"):
            out_lines.append(f"{INDENT}{ln}" if ln else "")

    return "\n".join(out_lines).rstrip()


def render_bullet_list(node: Node, ctx: RenderContext) -> str:
    items = node.get("content", []) or []
    lines: list[str] = []
    ctx.list_depth += 1
    try:
        for item in items:
            blocks = _render_list_item_inner(item, ctx)
            lines.append(_join_list_item("- ", blocks))
    finally:
        ctx.list_depth -= 1
    return "\n".join(lines)


def render_ordered_list(node: Node, ctx: RenderContext) -> str:
    items = node.get("content", []) or []
    start = int(node.get("attrs", {}).get("order", 1) or 1)
    lines: list[str] = []
    ctx.list_depth += 1
    try:
        for i, item in enumerate(items):
            blocks = _render_list_item_inner(item, ctx)
            lines.append(_join_list_item(f"{start + i}. ", blocks))
    finally:
        ctx.list_depth -= 1
    return "\n".join(lines)


def render_check_list(node: Node, ctx: RenderContext) -> str:
    items = node.get("content", []) or []
    lines: list[str] = []
    ctx.list_depth += 1
    try:
        for item in items:
            checked = bool(item.get("attrs", {}).get("checked", False))
            mark = "[x]" if checked else "[ ]"
            blocks = _render_list_item_inner(item, ctx)
            lines.append(_join_list_item(f"- {mark} ", blocks))
    finally:
        ctx.list_depth -= 1
    return "\n".join(lines)


RENDERERS = {
    "bullet_list": render_bullet_list,
    "ordered_list": render_ordered_list,
    "check_list": render_check_list,
}
