from __future__ import annotations

from boxnote2md.render import RenderContext, render_document


def text(s, marks=None):
    n = {"type": "text", "text": s}
    if marks:
        n["marks"] = marks
    return n


def para(*children):
    return {"type": "paragraph", "content": list(children)}


def cell(*content, colspan=1, rowspan=1, colwidth=None):
    attrs = {"colspan": colspan, "rowspan": rowspan, "colwidth": colwidth}
    return {"type": "table_cell", "attrs": attrs, "content": list(content)}


def row(*cells):
    return {"type": "table_row", "content": list(cells)}


def doc(*blocks):
    return {"type": "doc", "content": list(blocks)}


def test_simple_3x3_table():
    md = render_document(
        doc(
            {
                "type": "table",
                "content": [
                    row(cell(para(text("a"))), cell(para(text("b"))), cell(para(text("c")))),
                    row(cell(para(text("d"))), cell(para(text("e"))), cell(para(text("f")))),
                ],
            }
        ),
        RenderContext(),
    )
    assert md == (
        "| a | b | c |\n"
        "| --- | --- | --- |\n"
        "| d | e | f |\n"
    )


def test_table_inline_marks_in_cells():
    md = render_document(
        doc(
            {
                "type": "table",
                "content": [
                    row(
                        cell(para(text("h1"))),
                        cell(para(text("h2"))),
                    ),
                    row(
                        cell(para(text("bold", marks=[{"type": "strong"}]))),
                        cell(para(text("plain"))),
                    ),
                ],
            }
        ),
        RenderContext(),
    )
    assert md == (
        "| h1 | h2 |\n"
        "| --- | --- |\n"
        "| **bold** | plain |\n"
    )


def test_table_pipe_escape_and_newline_replacement():
    md = render_document(
        doc(
            {
                "type": "table",
                "content": [
                    row(cell(para(text("a|b"))), cell(para(text("c\nd")))),
                ],
            }
        ),
        RenderContext(),
    )
    assert "a\\|b" in md
    assert "c<br>d" in md


def test_table_multiple_paragraphs_in_cell():
    md = render_document(
        doc(
            {
                "type": "table",
                "content": [
                    row(cell(para(text("p1")), para(text("p2")))),
                ],
            }
        ),
        RenderContext(),
    )
    assert "p1<br><br>p2" in md


def test_table_colspan_warning_but_renders():
    ctx = RenderContext()
    md = render_document(
        doc(
            {
                "type": "table",
                "content": [
                    row(cell(para(text("a")), colspan=2), cell(para(text("b")))),
                    row(cell(para(text("c"))), cell(para(text("d"))), cell(para(text("e")))),
                ],
            }
        ),
        ctx,
    )
    assert any("colspan" in w for w in ctx.warnings)
    # 列数は最大行 (3) に揃えられる、欠損はパディング
    assert md.startswith("| a | b |  |\n")


def test_empty_paragraph_in_cell():
    md = render_document(
        doc(
            {
                "type": "table",
                "content": [
                    row(cell(para(text("a"))), cell({"type": "paragraph"})),
                ],
            }
        ),
        RenderContext(),
    )
    assert md == (
        "| a |  |\n"
        "| --- | --- |\n"
    )
