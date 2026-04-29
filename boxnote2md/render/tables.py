"""table / table_row / table_cell の Markdown 化。

GFM テーブルを出力。先頭行をヘッダ扱い。セル内の改行は <br> に、パイプは \\| に
エスケープ。colspan/rowspan は警告を出して 1×1 として扱う。
"""

from __future__ import annotations

from boxnote2md.render.document import Node, RenderContext, render_block, render_children_inline


def _render_cell(cell: Node, ctx: RenderContext) -> str:
    """table_cell の中身をセル1個分の文字列にする。

    複数段落は <br><br> で結合。リスト等が入る場合は render_block を経由しつつ
    最後に改行を <br> に置換する。
    """
    prev = ctx.in_table_cell
    ctx.in_table_cell = True
    try:
        parts: list[str] = []
        for child in cell.get("content", []) or []:
            ct = child.get("type")
            if ct == "paragraph":
                parts.append(render_children_inline(child, ctx))
            else:
                parts.append(render_block(child, ctx))
        body = "<br><br>".join(p for p in parts if p)
    finally:
        ctx.in_table_cell = prev

    # セル内の生改行を <br> へ
    body = body.replace("\r\n", "\n").replace("\n", "<br>")
    # パイプエスケープ
    body = body.replace("|", "\\|")
    return body


def render_table(node: Node, ctx: RenderContext) -> str:
    rows_raw = [r for r in node.get("content", []) or [] if r.get("type") == "table_row"]
    if not rows_raw:
        return ""

    # 各行をセル文字列のリストに変換しつつ、colspan/rowspan を警告
    grid: list[list[str]] = []
    for row in rows_raw:
        row_cells: list[str] = []
        for cell in row.get("content", []) or []:
            if cell.get("type") != "table_cell":
                continue
            attrs = cell.get("attrs", {}) or {}
            cs = int(attrs.get("colspan") or 1)
            rs = int(attrs.get("rowspan") or 1)
            if cs > 1 or rs > 1:
                ctx.warn(
                    f"table cell has colspan={cs} rowspan={rs} — "
                    "rendered as a single 1x1 cell (Markdown does not support span)"
                )
            row_cells.append(_render_cell(cell, ctx))
        grid.append(row_cells)

    # 列数は最大値で揃える
    col_count = max(len(r) for r in grid)
    for r in grid:
        while len(r) < col_count:
            r.append("")

    # GFM 出力
    header = grid[0]
    lines = ["| " + " | ".join(header) + " |"]
    lines.append("| " + " | ".join(["---"] * col_count) + " |")
    for r in grid[1:]:
        lines.append("| " + " | ".join(r) + " |")
    return "\n".join(lines)


RENDERERS = {
    "table": render_table,
}
