from __future__ import annotations

from boxnote2md.render import RenderContext, render_document


def text(s, marks=None):
    n = {"type": "text", "text": s}
    if marks:
        n["marks"] = marks
    return n


def para(*children):
    return {"type": "paragraph", "content": list(children)}


def doc(*blocks):
    return {"type": "doc", "content": list(blocks)}


def test_blockquote_single_paragraph():
    md = render_document(
        doc({"type": "blockquote", "content": [para(text("hello"))]}),
        RenderContext(),
    )
    assert md == "> hello\n"


def test_blockquote_multi_paragraph():
    md = render_document(
        doc(
            {
                "type": "blockquote",
                "content": [para(text("first")), para(text("second"))],
            }
        ),
        RenderContext(),
    )
    assert md == "> first\n>\n> second\n"


def test_code_block_with_language():
    md = render_document(
        doc(
            {
                "type": "code_block",
                "attrs": {"language": "Python"},
                "content": [text('import json\nprint("hi")')],
            }
        ),
        RenderContext(),
    )
    assert md == '```python\nimport json\nprint("hi")\n```\n'


def test_code_block_without_language():
    md = render_document(
        doc({"type": "code_block", "content": [text("plain")]}),
        RenderContext(),
    )
    assert md == "```\nplain\n```\n"


def test_code_block_does_not_escape_backticks():
    md = render_document(
        doc(
            {
                "type": "code_block",
                "attrs": {"language": "sh"},
                "content": [text("echo `date`")],
            }
        ),
        RenderContext(),
    )
    assert md == "```sh\necho `date`\n```\n"


def test_call_out_box_default_renders_as_quote():
    md = render_document(
        doc(
            {
                "type": "call_out_box",
                "attrs": {"emoji": "⚠️", "backgroundColor": "#51ce9a"},
                "content": [para(text("warning"))],
            }
        ),
        RenderContext(),
    )
    assert md == "> ⚠️ warning\n"


def test_call_out_box_keep_styles_renders_div():
    md = render_document(
        doc(
            {
                "type": "call_out_box",
                "attrs": {"emoji": "💡", "backgroundColor": "#ff0"},
                "content": [para(text("tip"))],
            }
        ),
        RenderContext(keep_styles=True),
    )
    assert "background-color:#ff0" in md
    assert "💡 tip" in md


def test_call_out_box_no_emoji():
    md = render_document(
        doc(
            {
                "type": "call_out_box",
                "attrs": {"backgroundColor": "#51ce9a"},
                "content": [para(text("plain callout"))],
            }
        ),
        RenderContext(),
    )
    assert md == "> plain callout\n"
