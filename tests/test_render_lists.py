from __future__ import annotations

from boxnote2md.render import RenderContext, render_document


def text(s, marks=None):
    n = {"type": "text", "text": s}
    if marks:
        n["marks"] = marks
    return n


def para(*children):
    return {"type": "paragraph", "content": list(children)}


def li(*content):
    return {"type": "list_item", "content": list(content)}


def cli_item(checked, *content):
    return {"type": "check_list_item", "attrs": {"checked": checked}, "content": list(content)}


def doc(*blocks):
    return {"type": "doc", "content": list(blocks)}


def test_bullet_list_simple():
    md = render_document(
        doc({"type": "bullet_list", "content": [li(para(text("a"))), li(para(text("b")))]}),
        RenderContext(),
    )
    assert md == "- a\n- b\n"


def test_ordered_list_with_order():
    md = render_document(
        doc(
            {
                "type": "ordered_list",
                "attrs": {"order": 3},
                "content": [li(para(text("x"))), li(para(text("y")))],
            }
        ),
        RenderContext(),
    )
    assert md == "3. x\n4. y\n"


def test_check_list_mixed():
    md = render_document(
        doc(
            {
                "type": "check_list",
                "content": [
                    cli_item(True, para(text("done"))),
                    cli_item(False, para(text("todo"))),
                ],
            }
        ),
        RenderContext(),
    )
    assert md == "- [x] done\n- [ ] todo\n"


def test_bullet_list_with_inline_marks():
    md = render_document(
        doc(
            {
                "type": "bullet_list",
                "content": [li(para(text("a", marks=[{"type": "strong"}])))],
            }
        ),
        RenderContext(),
    )
    assert md == "- **a**\n"


def test_nested_bullet_list():
    md = render_document(
        doc(
            {
                "type": "bullet_list",
                "content": [
                    li(
                        para(text("outer")),
                        {
                            "type": "bullet_list",
                            "content": [li(para(text("inner1"))), li(para(text("inner2")))],
                        },
                    ),
                    li(para(text("outer2"))),
                ],
            }
        ),
        RenderContext(),
    )
    assert md == (
        "- outer\n"
        "\n"
        "    - inner1\n"
        "    - inner2\n"
        "- outer2\n"
    )


def test_ordered_default_start_one():
    md = render_document(
        doc(
            {
                "type": "ordered_list",
                "content": [li(para(text("a")))],
            }
        ),
        RenderContext(),
    )
    assert md == "1. a\n"
