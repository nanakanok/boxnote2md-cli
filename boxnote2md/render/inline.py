"""インラインノードとマークの Markdown 化。

text ノードに `marks: [{type, attrs}]` が付く。
出力時は外側から内側へ次の順で wrap する (閉じタグの整合と読みやすさ重視):
  link → strong → em → underline → strikethrough → highlight
"""

from __future__ import annotations

from typing import Any

from boxnote2md.render.document import Node, RenderContext

# 出力で wrap するマーク (内側 → 外側 の順で適用しても良いように、後段で逆順反転)
_FORMAT_ORDER = ["highlight", "strikethrough", "underline", "em", "strong", "link"]

# Markdown / HTML への wrap 規則。
_WRAPPERS = {
    "strong": ("**", "**"),
    "em": ("*", "*"),
    "underline": ("<u>", "</u>"),
    "strikethrough": ("~~", "~~"),
}


def _escape_markdown(text: str) -> str:
    """段落内のテキストで誤って MD 記法と解釈されないようエスケープ。

    控えめに: 行頭の `#`, `-`, `>`, `*`, `+`, `1.` 系は段落整形側で扱う想定。
    ここではバックスラッシュ・パイプ・バッククォートのみを最小限処理する。
    """
    return text.replace("\\", "\\\\").replace("`", "\\`")


def render_text_with_marks(node: Node, ctx: RenderContext) -> str:
    text = node.get("text", "")
    if not text:
        return ""

    rendered = _escape_markdown(text)

    marks_by_type: dict[str, dict[str, Any]] = {}
    for m in node.get("marks", []) or []:
        marks_by_type[m["type"]] = m

    # presentational marks (keep_styles=False では落とす)
    for k in ("font_size", "font_color", "highlight"):
        if k in marks_by_type and not ctx.keep_styles:
            marks_by_type.pop(k)

    # presentational marks (keep_styles=True では HTML span)
    if ctx.keep_styles:
        style_parts: list[str] = []
        if "font_size" in marks_by_type:
            size = marks_by_type["font_size"].get("attrs", {}).get("size")
            if size:
                style_parts.append(f"font-size:{size}")
            marks_by_type.pop("font_size")
        if "font_color" in marks_by_type:
            color = marks_by_type["font_color"].get("attrs", {}).get("color")
            if color:
                style_parts.append(f"color:{color}")
            marks_by_type.pop("font_color")
        if "highlight" in marks_by_type:
            color = marks_by_type["highlight"].get("attrs", {}).get("color")
            if color:
                style_parts.append(f"background-color:{color}")
            marks_by_type.pop("highlight")
        if style_parts:
            rendered = f'<span style="{";".join(style_parts)}">{rendered}</span>'

    # 標準マーク wrap (内側から外側の順)
    for mark_type in _FORMAT_ORDER:
        if mark_type not in marks_by_type:
            continue
        if mark_type == "link":
            href = marks_by_type["link"].get("attrs", {}).get("href", "")
            rendered = f"[{rendered}]({href})"
        elif mark_type in _WRAPPERS:
            left, right = _WRAPPERS[mark_type]
            # 既に空文字なら wrap しない (空 ** ** を避ける)
            if rendered:
                rendered = f"{left}{rendered}{right}"

    return rendered


def render_hard_break(_node: Node, ctx: RenderContext) -> str:
    # ProseMirror の hard_break (Box でも稀に出る可能性)
    return "<br>" if ctx.in_table_cell else "  \n"


RENDERERS = {
    "text": render_text_with_marks,
    "hard_break": render_hard_break,
}
