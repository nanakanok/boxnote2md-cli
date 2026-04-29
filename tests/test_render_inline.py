from __future__ import annotations

from boxnote2md.render import RenderContext, render_document


def doc_with(blocks):
    return {"type": "doc", "content": blocks}


def text(s, marks=None):
    n = {"type": "text", "text": s}
    if marks:
        n["marks"] = marks
    return n


def para(*children):
    return {"type": "paragraph", "content": list(children)}


def test_plain_paragraph():
    md = render_document(doc_with([para(text("hello"))]), RenderContext())
    assert md == "hello\n"


def test_strong_em_underline_strike():
    blocks = [
        para(text("a", marks=[{"type": "strong"}])),
        para(text("b", marks=[{"type": "em"}])),
        para(text("c", marks=[{"type": "underline"}])),
        para(text("d", marks=[{"type": "strikethrough"}])),
    ]
    md = render_document(doc_with(blocks), RenderContext())
    assert md == "**a**\n\n*b*\n\n<u>c</u>\n\n~~d~~\n"


def test_bold_italic_combination():
    blocks = [para(text("be", marks=[{"type": "em"}, {"type": "strong"}]))]
    md = render_document(doc_with(blocks), RenderContext())
    # 内側 em → 外側 strong
    assert md == "***be***\n"


def test_link():
    blocks = [
        para(
            text("see "),
            text("here", marks=[{"type": "link", "attrs": {"href": "https://example.com"}}]),
        )
    ]
    md = render_document(doc_with(blocks), RenderContext())
    assert md == "see [here](https://example.com)\n"


def test_link_with_strong():
    blocks = [
        para(
            text(
                "x",
                marks=[
                    {"type": "strong"},
                    {"type": "link", "attrs": {"href": "https://e.com"}},
                ],
            )
        )
    ]
    md = render_document(doc_with(blocks), RenderContext())
    # link が一番外側に来る
    assert md == "[**x**](https://e.com)\n"


def test_heading_levels():
    blocks = [
        {"type": "heading", "attrs": {"level": 1}, "content": [text("H1")]},
        {"type": "heading", "attrs": {"level": 3}, "content": [text("H3")]},
    ]
    md = render_document(doc_with(blocks), RenderContext())
    assert md == "# H1\n\n### H3\n"


def test_horizontal_rule():
    md = render_document(
        doc_with([para(text("a")), {"type": "horizontal_rule"}, para(text("b"))]),
        RenderContext(),
    )
    assert md == "a\n\n---\n\nb\n"


def test_presentational_marks_dropped_by_default():
    blocks = [
        para(
            text(
                "fancy",
                marks=[
                    {"type": "font_size", "attrs": {"size": "1.5em"}},
                    {"type": "font_color", "attrs": {"color": "#ff0000"}},
                    {"type": "highlight", "attrs": {"color": "#ff0"}},
                ],
            )
        )
    ]
    md = render_document(doc_with(blocks), RenderContext(keep_styles=False))
    assert md == "fancy\n"


def test_presentational_marks_kept_with_flag():
    blocks = [
        para(
            text(
                "fancy",
                marks=[{"type": "font_color", "attrs": {"color": "#ff0000"}}],
            )
        )
    ]
    md = render_document(doc_with(blocks), RenderContext(keep_styles=True))
    assert md == '<span style="color:#ff0000">fancy</span>\n'


def test_author_id_and_annotation_id_ignored():
    blocks = [
        para(
            text(
                "x",
                marks=[
                    {"type": "author_id", "attrs": {"authorId": "1"}},
                    {"type": "annotation_id", "attrs": {"annotationId": "a"}},
                    {"type": "strong"},
                ],
            )
        )
    ]
    md = render_document(doc_with(blocks), RenderContext())
    assert md == "**x**\n"


def test_empty_paragraph_collapsed():
    blocks = [para(text("a")), {"type": "paragraph"}, para(text("b"))]
    md = render_document(doc_with(blocks), RenderContext())
    assert md == "a\n\nb\n"
