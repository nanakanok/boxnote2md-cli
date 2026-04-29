"""Markdown → ProseMirror `doc` ノード パーサ。

仕様の詳細は planning_md2boxnote.md を参照。
標準ライブラリのみで動く実装。CommonMark 厳密準拠ではないが、本ツール
(boxnote2md) が出力する MD のラウンドトリップは確実に通せるサブセットを実装する。
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Any

Node = dict[str, Any]


# --- 正規表現 (block 判定用) ---
RE_HEADING = re.compile(r"^(#{1,6})[ \t]+(.*?)\s*#*\s*$")
RE_HR = re.compile(
    r"^[ \t]{0,3}(?:-[ \t]*){3,}$|^[ \t]{0,3}(?:\*[ \t]*){3,}$|^[ \t]{0,3}(?:_[ \t]*){3,}$"
)
RE_FENCE_OPEN = re.compile(r"^([ \t]{0,3})(```+|~~~+)[ \t]*([^\s`]*)[ \t]*$")
RE_BLOCKQUOTE = re.compile(r"^[ \t]{0,3}>[ \t]?(.*)$")
RE_BULLET = re.compile(r"^([ \t]*)([-*+])[ \t]+(.*)$")
RE_ORDERED = re.compile(r"^([ \t]*)(\d+)[.)][ \t]+(.*)$")
RE_CHECKBOX = re.compile(r"^\[([ xX])\][ \t]+(.*)$")
RE_TABLE_SEP = re.compile(
    r"^[ \t]*\|[ \t]*:?-{3,}:?[ \t]*(\|[ \t]*:?-{3,}:?[ \t]*)*\|[ \t]*$"
)
RE_TABLE_ROW = re.compile(r"^[ \t]*\|.*\|[ \t]*$")


def parse_markdown(text: str) -> Node:
    """Markdown 全体をパースし、ProseMirror `doc` ノードを返す。"""
    if text.startswith("﻿"):
        text = text[1:]
    lines = text.replace("\r\n", "\n").replace("\r", "\n").split("\n")
    blocks = parse_blocks(lines, 0, len(lines))
    return {
        "type": "doc",
        "attrs": {"table_of_contents": {"enabled": False, "allowedLevels": [1, 2, 3]}},
        "content": blocks,
    }


# ============================================================
# ブロック層
# ============================================================


def parse_blocks(lines: list[str], start: int, end: int) -> list[Node]:
    out: list[Node] = []
    i = start
    while i < end:
        line = lines[i]

        if not line.strip():
            i += 1
            continue

        m = RE_HEADING.match(line)
        if m:
            level = len(m.group(1))
            content = m.group(2).strip()
            out.append(
                {
                    "type": "heading",
                    "attrs": {"level": level},
                    "content": parse_inline(content) if content else [],
                }
            )
            i += 1
            continue

        if RE_HR.match(line):
            out.append({"type": "horizontal_rule"})
            i += 1
            continue

        m = RE_FENCE_OPEN.match(line)
        if m:
            i = _consume_code_block(lines, i, end, m, out)
            continue

        if RE_BLOCKQUOTE.match(line):
            i = _consume_blockquote(lines, i, end, out)
            continue

        if RE_TABLE_ROW.match(line) and i + 1 < end and RE_TABLE_SEP.match(lines[i + 1]):
            table_node, consumed = parse_table(lines, i, end)
            out.append(table_node)
            i = consumed
            continue

        if RE_BULLET.match(line) or RE_ORDERED.match(line):
            list_node, consumed = parse_list(lines, i, end)
            out.append(list_node)
            i = consumed
            continue

        # 段落
        para_lines: list[str] = []
        while i < end:
            ln = lines[i]
            if not ln.strip():
                break
            if (
                RE_HEADING.match(ln)
                or RE_HR.match(ln)
                or RE_FENCE_OPEN.match(ln)
                or RE_BLOCKQUOTE.match(ln)
                or RE_BULLET.match(ln)
                or RE_ORDERED.match(ln)
            ):
                break
            if RE_TABLE_ROW.match(ln) and i + 1 < end and RE_TABLE_SEP.match(lines[i + 1]):
                break
            para_lines.append(ln)
            i += 1
        if para_lines:
            text = "\n".join(para_lines).strip()
            if text:
                out.append({"type": "paragraph", "content": parse_inline(text)})
    return out


def _consume_code_block(
    lines: list[str], start: int, end: int, open_match: re.Match, out: list[Node]
) -> int:
    indent = len(open_match.group(1))
    fence = open_match.group(2)
    language = open_match.group(3).strip()
    i = start + 1
    code_lines: list[str] = []
    fence_char = fence[0]
    fence_len = len(fence)
    close_re = re.compile(
        r"^[ \t]{0,3}" + re.escape(fence_char) + r"{" + str(fence_len) + r",}\s*$"
    )
    while i < end and not close_re.match(lines[i]):
        ln = lines[i]
        if indent and ln[:indent].strip() == "":
            ln = ln[indent:]
        code_lines.append(ln)
        i += 1
    if i < end:
        i += 1
    text = "\n".join(code_lines)
    node: Node = {"type": "code_block", "attrs": {"language": language or ""}}
    if text:
        node["content"] = [{"type": "text", "text": text}]
    out.append(node)
    return i


def _consume_blockquote(lines: list[str], start: int, end: int, out: list[Node]) -> int:
    j = start
    stripped: list[str] = []
    while j < end:
        m2 = RE_BLOCKQUOTE.match(lines[j])
        if m2:
            stripped.append(m2.group(1))
            j += 1
        elif lines[j].strip() == "":
            break
        else:
            # lazy continuation
            stripped.append(lines[j])
            j += 1
    inner = parse_blocks(stripped, 0, len(stripped))
    out.append({"type": "blockquote", "content": inner})
    return j


# ============================================================
# リスト
# ============================================================


@dataclass
class _ListItemRaw:
    indent: int
    marker_width: int
    is_ordered: bool
    order: int
    is_check: bool
    checked: bool
    content_lines: list[str]


def _classify_list_marker(line: str):
    """戻り値: (indent, marker_width, is_ordered, order, rest) | None"""
    m = RE_BULLET.match(line)
    if m:
        prefix = m.group(1)
        marker = m.group(2)
        rest = m.group(3)
        indent = len(prefix.expandtabs(4))
        marker_pos = len(prefix)
        body_pos = marker_pos + len(marker)
        # マーカー後の空白を marker_width に含める
        while body_pos < len(line) and line[body_pos] in " \t":
            body_pos += 1
        marker_width = body_pos - marker_pos
        return indent, marker_width, False, 0, rest
    m = RE_ORDERED.match(line)
    if m:
        prefix = m.group(1)
        order_str = m.group(2)
        rest = m.group(3)
        indent = len(prefix.expandtabs(4))
        marker_pos = len(prefix)
        body_pos = marker_pos + len(order_str) + 1  # 数字 + '.' or ')'
        while body_pos < len(line) and line[body_pos] in " \t":
            body_pos += 1
        marker_width = body_pos - marker_pos
        return indent, marker_width, True, int(order_str), rest
    return None


def parse_list(lines: list[str], start: int, end: int) -> tuple[Node, int]:
    first = _classify_list_marker(lines[start])
    assert first is not None
    base_indent, _, base_ordered, base_order, _ = first

    items: list[_ListItemRaw] = []
    i = start
    while i < end:
        line = lines[i]
        cls = _classify_list_marker(line)
        if cls is None or cls[0] != base_indent or cls[2] != base_ordered:
            break
        indent, marker_width, is_ordered, order, rest = cls

        cm = RE_CHECKBOX.match(rest)
        if cm:
            is_check = True
            checked = cm.group(1) in ("x", "X")
            rest = cm.group(2)
        else:
            is_check = False
            checked = False

        item = _ListItemRaw(
            indent=indent,
            marker_width=marker_width,
            is_ordered=is_ordered,
            order=order,
            is_check=is_check,
            checked=checked,
            content_lines=[rest],
        )
        i += 1
        cont_indent = indent + marker_width

        while i < end:
            ln = lines[i]
            if not ln.strip():
                # 空行: 後続が継続できれば吸収
                if i + 1 < end:
                    nxt = lines[i + 1]
                    nxt_cls = _classify_list_marker(nxt)
                    if nxt_cls is not None and nxt_cls[0] == base_indent:
                        break
                    nxt_ws = len(nxt) - len(nxt.lstrip(" \t"))
                    if nxt.strip() and nxt_ws >= cont_indent:
                        item.content_lines.append("")
                        i += 1
                        continue
                break
            ln_cls = _classify_list_marker(ln)
            if ln_cls is not None and ln_cls[0] == base_indent:
                break
            ws = len(ln) - len(ln.lstrip(" \t"))
            if ws >= cont_indent or ln_cls is not None:
                item.content_lines.append(ln[cont_indent:] if ws >= cont_indent else ln)
                i += 1
                continue
            # lazy continuation
            item.content_lines.append(ln.lstrip())
            i += 1

        items.append(item)

    # check_list 判定: 1項目目が check ならリスト全体
    is_check_list = bool(items) and items[0].is_check

    item_nodes: list[Node] = []
    for it in items:
        sub_blocks = parse_blocks(it.content_lines, 0, len(it.content_lines))
        if not sub_blocks:
            sub_blocks = [{"type": "paragraph"}]
        if is_check_list:
            item_nodes.append(
                {
                    "type": "check_list_item",
                    "attrs": {"checked": it.checked},
                    "content": sub_blocks,
                }
            )
        else:
            item_nodes.append({"type": "list_item", "content": sub_blocks})

    if is_check_list:
        return {"type": "check_list", "content": item_nodes}, i
    if base_ordered:
        return (
            {"type": "ordered_list", "attrs": {"order": base_order}, "content": item_nodes},
            i,
        )
    return {"type": "bullet_list", "content": item_nodes}, i


# ============================================================
# テーブル
# ============================================================


def _split_table_row(line: str) -> list[str]:
    s = line.strip()
    if s.startswith("|"):
        s = s[1:]
    if s.endswith("|"):
        s = s[:-1]
    cells: list[str] = []
    buf: list[str] = []
    i = 0
    while i < len(s):
        c = s[i]
        if c == "\\" and i + 1 < len(s):
            buf.append(s[i + 1])
            i += 2
            continue
        if c == "|":
            cells.append("".join(buf).strip())
            buf = []
            i += 1
            continue
        buf.append(c)
        i += 1
    cells.append("".join(buf).strip())
    return cells


def parse_table(lines: list[str], start: int, end: int) -> tuple[Node, int]:
    header = _split_table_row(lines[start])
    col_count = len(header)
    i = start + 2
    rows: list[list[str]] = [header]
    while i < end:
        ln = lines[i]
        if not RE_TABLE_ROW.match(ln):
            break
        cells = _split_table_row(ln)
        if len(cells) < col_count:
            cells.extend([""] * (col_count - len(cells)))
        else:
            cells = cells[:col_count]
        rows.append(cells)
        i += 1

    table_rows: list[Node] = []
    for r in rows:
        cell_nodes: list[Node] = []
        for cell_text in r:
            paragraphs_text = (
                cell_text.split("<br><br>") if "<br><br>" in cell_text else [cell_text]
            )
            paragraphs: list[Node] = []
            for ptext in paragraphs_text:
                inner = ptext.replace("<br>", "\n")
                if inner.strip():
                    paragraphs.append({"type": "paragraph", "content": parse_inline(inner)})
                else:
                    paragraphs.append({"type": "paragraph"})
            cell_nodes.append(
                {
                    "type": "table_cell",
                    "attrs": {"colspan": 1, "rowspan": 1, "colwidth": None},
                    "content": paragraphs,
                }
            )
        table_rows.append({"type": "table_row", "content": cell_nodes})

    return {"type": "table", "content": table_rows}, i


# ============================================================
# インライン層
# ============================================================

T_TEXT = "text"
T_STRONG = "strong_marker"
T_EM = "em_marker"
T_STRIKE = "strike_marker"
T_U_OPEN = "u_open"
T_U_CLOSE = "u_close"
T_HARD_BREAK = "hard_break"
T_LINK = "link"
T_IMAGE = "image"


def parse_inline(text: str) -> list[Node]:
    return _flatten_tokens(tokenize_inline(text))


def tokenize_inline(text: str) -> list[tuple]:
    tokens: list[tuple] = []
    i = 0
    n = len(text)
    while i < n:
        c = text[i]
        # backslash escape
        if c == "\\" and i + 1 < n:
            tokens.append((T_TEXT, text[i + 1]))
            i += 2
            continue
        # hard break "  \n"
        if c == " " and text[i : i + 3] == "  \n":
            tokens.append((T_HARD_BREAK,))
            i += 3
            continue
        # image
        if c == "!" and i + 1 < n and text[i + 1] == "[":
            consumed = _try_consume_image(text, i)
            if consumed is not None:
                alt, url, end_pos = consumed
                tokens.append((T_IMAGE, alt, url))
                i = end_pos
                continue
        # link
        if c == "[":
            consumed = _try_consume_link(text, i)
            if consumed is not None:
                inner_tokens, url, end_pos = consumed
                tokens.append((T_LINK, inner_tokens, url))
                i = end_pos
                continue
        # <u>, </u>
        lower3 = text[i : i + 3].lower()
        lower4 = text[i : i + 4].lower()
        if lower3 == "<u>":
            tokens.append((T_U_OPEN,))
            i += 3
            continue
        if lower4 == "</u>":
            tokens.append((T_U_CLOSE,))
            i += 4
            continue
        # strong / em
        if text[i : i + 2] == "**":
            tokens.append((T_STRONG,))
            i += 2
            continue
        if c == "*":
            tokens.append((T_EM,))
            i += 1
            continue
        # _emphasis_ (単語境界のときのみ)
        if c == "_":
            prev_c = text[i - 1] if i > 0 else " "
            next_c = text[i + 1] if i + 1 < n else " "
            if not (prev_c.isalnum() and next_c.isalnum()):
                tokens.append((T_EM,))
                i += 1
                continue
        # ~~strike~~
        if text[i : i + 2] == "~~":
            tokens.append((T_STRIKE,))
            i += 2
            continue
        # plain text
        tokens.append((T_TEXT, c))
        i += 1
    return _coalesce_text(tokens)


def _coalesce_text(tokens: list[tuple]) -> list[tuple]:
    out: list[tuple] = []
    for tok in tokens:
        if tok[0] == T_TEXT and out and out[-1][0] == T_TEXT:
            out[-1] = (T_TEXT, out[-1][1] + tok[1])
        else:
            out.append(tok)
    return out


def _try_consume_image(text: str, start: int) -> tuple[str, str, int] | None:
    bracket_end = _find_matching_bracket(text, start + 1)
    if bracket_end < 0:
        return None
    if bracket_end + 1 >= len(text) or text[bracket_end + 1] != "(":
        return None
    alt = text[start + 2 : bracket_end]
    url, paren_end = _consume_url(text, bracket_end + 2)
    if url is None:
        return None
    return alt, url, paren_end + 1


def _try_consume_link(text: str, start: int) -> tuple[list[tuple], str, int] | None:
    bracket_end = _find_matching_bracket(text, start)
    if bracket_end < 0:
        return None
    if bracket_end + 1 >= len(text) or text[bracket_end + 1] != "(":
        return None
    label = text[start + 1 : bracket_end]
    url, paren_end = _consume_url(text, bracket_end + 2)
    if url is None:
        return None
    return tokenize_inline(label), url, paren_end + 1


def _find_matching_bracket(text: str, start_bracket_pos: int) -> int:
    """`text[start_bracket_pos]` が `[` の前提で対応する `]` の位置を返す。
    見つからなければ -1。
    """
    depth = 1
    i = start_bracket_pos + 1
    while i < len(text):
        c = text[i]
        if c == "\\" and i + 1 < len(text):
            i += 2
            continue
        if c == "[":
            depth += 1
        elif c == "]":
            depth -= 1
            if depth == 0:
                return i
        i += 1
    return -1


def _consume_url(text: str, start: int) -> tuple[str | None, int]:
    depth = 1
    i = start
    buf: list[str] = []
    while i < len(text) and depth > 0:
        c = text[i]
        if c == "\\" and i + 1 < len(text):
            buf.append(text[i + 1])
            i += 2
            continue
        if c == "(":
            depth += 1
            buf.append(c)
        elif c == ")":
            depth -= 1
            if depth == 0:
                break
            buf.append(c)
        else:
            buf.append(c)
        i += 1
    if depth != 0:
        return None, i
    return "".join(buf).strip(), i


def _classify_markers(tokens: list[tuple]) -> dict[int, tuple[str, bool]]:
    out: dict[int, tuple[str, bool]] = {}
    open_stacks: dict[str, list[int]] = {
        "strong": [],
        "em": [],
        "strikethrough": [],
        "underline": [],
    }
    for idx, tok in enumerate(tokens):
        kind = tok[0]
        if kind == T_STRONG:
            stk = open_stacks["strong"]
            if stk:
                out[stk.pop()] = ("strong", True)
                out[idx] = ("strong", False)
            else:
                stk.append(idx)
        elif kind == T_EM:
            stk = open_stacks["em"]
            if stk:
                out[stk.pop()] = ("em", True)
                out[idx] = ("em", False)
            else:
                stk.append(idx)
        elif kind == T_STRIKE:
            stk = open_stacks["strikethrough"]
            if stk:
                out[stk.pop()] = ("strikethrough", True)
                out[idx] = ("strikethrough", False)
            else:
                stk.append(idx)
        elif kind == T_U_OPEN:
            open_stacks["underline"].append(idx)
        elif kind == T_U_CLOSE:
            stk = open_stacks["underline"]
            if stk:
                out[stk.pop()] = ("underline", True)
                out[idx] = ("underline", False)
    return out


def _marker_literal(kind: str) -> str:
    return {
        T_STRONG: "**",
        T_EM: "*",
        T_STRIKE: "~~",
        T_U_OPEN: "<u>",
        T_U_CLOSE: "</u>",
    }.get(kind, "")


def _text_node(text: str, marks: list[dict]) -> Node:
    n: Node = {"type": "text", "text": text}
    if marks:
        n["marks"] = marks
    return n


def _basename(url: str) -> str:
    s = url.split("?", 1)[0].split("#", 1)[0]
    return s.rsplit("/", 1)[-1]


def _flatten_tokens(tokens: list[tuple]) -> list[Node]:
    classify = _classify_markers(tokens)
    nodes: list[Node] = []
    stack: list[str] = []

    def cur_marks() -> list[dict]:
        return [{"type": m} for m in stack]

    for idx, tok in enumerate(tokens):
        kind = tok[0]
        if kind == T_TEXT:
            if tok[1]:
                nodes.append(_text_node(tok[1], cur_marks()))
        elif kind == T_HARD_BREAK:
            nodes.append({"type": "hard_break"})
        elif kind == T_IMAGE:
            alt, url = tok[1], tok[2]
            file_name = _basename(url) or alt or "image"
            nodes.append(
                {
                    "type": "image",
                    "attrs": {
                        "src": url,
                        "alt": alt,
                        "title": "",
                        "boxSharedLink": "",
                        "boxFileId": "",
                        "fileName": file_name,
                        "placeholderState": "",
                        "width": None,
                        "height": None,
                    },
                }
            )
        elif kind == T_LINK:
            inner_tokens, url = tok[1], tok[2]
            inner_nodes = _flatten_tokens(inner_tokens)
            outer = cur_marks()
            for n in inner_nodes:
                if n.get("type") == "text":
                    marks = list(n.get("marks", []))
                    for m in outer:
                        if m not in marks:
                            marks.append(m)
                    marks.append({"type": "link", "attrs": {"href": url}})
                    n["marks"] = marks
                nodes.append(n)
        elif kind in (T_STRONG, T_EM, T_STRIKE, T_U_OPEN, T_U_CLOSE):
            role = classify.get(idx)
            if role is None:
                nodes.append(_text_node(_marker_literal(kind), cur_marks()))
            else:
                mark_type, opening = role
                if opening:
                    stack.append(mark_type)
                else:
                    if mark_type in stack:
                        stack.remove(mark_type)
    return _merge_adjacent_text(nodes)


def _merge_adjacent_text(nodes: list[Node]) -> list[Node]:
    out: list[Node] = []
    for n in nodes:
        if (
            out
            and n.get("type") == "text"
            and out[-1].get("type") == "text"
            and n.get("marks", []) == out[-1].get("marks", [])
        ):
            out[-1]["text"] = out[-1].get("text", "") + n.get("text", "")
        else:
            out.append(n)
    return out
