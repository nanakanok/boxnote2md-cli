from __future__ import annotations

from boxnote2md.md.parser import parse_inline


def text(s, marks=None):
    n = {"type": "text", "text": s}
    if marks:
        n["marks"] = marks
    return n


def test_plain():
    assert parse_inline("hello world") == [text("hello world")]


def test_strong():
    assert parse_inline("**bold**") == [text("bold", marks=[{"type": "strong"}])]


def test_em():
    assert parse_inline("*it*") == [text("it", marks=[{"type": "em"}])]


def test_strong_em_combo():
    # ***x*** = ** + * + x + * + **  -> em then strong (innermost em first)
    res = parse_inline("***x***")
    assert res == [text("x", marks=[{"type": "strong"}, {"type": "em"}])]


def test_underline():
    assert parse_inline("<u>x</u>") == [text("x", marks=[{"type": "underline"}])]


def test_strike():
    assert parse_inline("~~x~~") == [text("x", marks=[{"type": "strikethrough"}])]


def test_link():
    res = parse_inline("[label](https://e.com)")
    assert res == [
        text("label", marks=[{"type": "link", "attrs": {"href": "https://e.com"}}])
    ]


def test_link_with_strong_inside():
    res = parse_inline("[**x**](https://e.com)")
    assert res == [
        text(
            "x",
            marks=[
                {"type": "strong"},
                {"type": "link", "attrs": {"href": "https://e.com"}},
            ],
        )
    ]


def test_image():
    res = parse_inline("![alt](https://e.com/x.png)")
    assert len(res) == 1
    img = res[0]
    assert img["type"] == "image"
    assert img["attrs"]["src"] == "https://e.com/x.png"
    assert img["attrs"]["alt"] == "alt"
    assert img["attrs"]["fileName"] == "x.png"


def test_unmatched_marker_is_literal():
    res = parse_inline("a**b")
    # ** がペアにならないので literal
    assert res == [text("a**b")]


def test_mixed_text_and_marks():
    res = parse_inline("hello **world** end")
    assert res == [
        text("hello "),
        text("world", marks=[{"type": "strong"}]),
        text(" end"),
    ]


def test_escape_backslash():
    assert parse_inline(r"\*not em\*") == [text("*not em*")]


def test_hard_break():
    res = parse_inline("a  \nb")
    assert res == [text("a"), {"type": "hard_break"}, text("b")]


def test_link_url_with_parens():
    res = parse_inline("[x](http://e.com/path(1))")
    assert res == [
        text("x", marks=[{"type": "link", "attrs": {"href": "http://e.com/path(1)"}}])
    ]
