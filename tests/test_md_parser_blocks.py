from __future__ import annotations

from boxnote2md.md.parser import parse_markdown


def test_paragraph():
    doc = parse_markdown("hello world")
    assert doc["content"] == [
        {"type": "paragraph", "content": [{"type": "text", "text": "hello world"}]}
    ]


def test_heading_levels():
    doc = parse_markdown("# H1\n\n### H3")
    assert doc["content"] == [
        {"type": "heading", "attrs": {"level": 1},
         "content": [{"type": "text", "text": "H1"}]},
        {"type": "heading", "attrs": {"level": 3},
         "content": [{"type": "text", "text": "H3"}]},
    ]


def test_horizontal_rule():
    doc = parse_markdown("a\n\n---\n\nb")
    types = [n["type"] for n in doc["content"]]
    assert types == ["paragraph", "horizontal_rule", "paragraph"]


def test_blockquote_single():
    doc = parse_markdown("> hello")
    assert doc["content"] == [
        {
            "type": "blockquote",
            "content": [
                {"type": "paragraph", "content": [{"type": "text", "text": "hello"}]}
            ],
        }
    ]


def test_blockquote_multi_paragraph():
    doc = parse_markdown("> first\n>\n> second")
    bq = doc["content"][0]
    assert bq["type"] == "blockquote"
    assert len(bq["content"]) == 2


def test_code_block_with_language():
    doc = parse_markdown("```python\nimport json\nprint('hi')\n```")
    assert doc["content"] == [
        {
            "type": "code_block",
            "attrs": {"language": "python"},
            "content": [{"type": "text", "text": "import json\nprint('hi')"}],
        }
    ]


def test_code_block_empty():
    doc = parse_markdown("```\n```")
    assert doc["content"] == [{"type": "code_block", "attrs": {"language": ""}}]


def test_bullet_list():
    doc = parse_markdown("- a\n- b")
    assert doc["content"] == [
        {
            "type": "bullet_list",
            "content": [
                {
                    "type": "list_item",
                    "content": [
                        {"type": "paragraph", "content": [{"type": "text", "text": "a"}]}
                    ],
                },
                {
                    "type": "list_item",
                    "content": [
                        {"type": "paragraph", "content": [{"type": "text", "text": "b"}]}
                    ],
                },
            ],
        }
    ]


def test_ordered_list_with_start():
    doc = parse_markdown("3. x\n4. y")
    node = doc["content"][0]
    assert node["type"] == "ordered_list"
    assert node["attrs"]["order"] == 3
    assert len(node["content"]) == 2


def test_check_list():
    doc = parse_markdown("- [x] done\n- [ ] todo")
    node = doc["content"][0]
    assert node["type"] == "check_list"
    assert node["content"][0]["attrs"]["checked"] is True
    assert node["content"][1]["attrs"]["checked"] is False


def test_nested_bullet_list():
    md = "- outer\n\n    - inner1\n    - inner2\n- outer2"
    doc = parse_markdown(md)
    bl = doc["content"][0]
    assert bl["type"] == "bullet_list"
    assert len(bl["content"]) == 2
    inner = bl["content"][0]["content"]
    # inner: 段落 + ネスト bullet_list
    types = [n["type"] for n in inner]
    assert "bullet_list" in types


def test_table_simple():
    md = "| h1 | h2 |\n| --- | --- |\n| a | b |"
    doc = parse_markdown(md)
    t = doc["content"][0]
    assert t["type"] == "table"
    assert len(t["content"]) == 2  # ヘッダ + データ1行
    row0 = t["content"][0]
    assert row0["type"] == "table_row"
    cell0 = row0["content"][0]
    assert cell0["type"] == "table_cell"
    assert cell0["attrs"]["colspan"] == 1


def test_table_with_inline_marks():
    md = "| h |\n| --- |\n| **bold** |"
    doc = parse_markdown(md)
    t = doc["content"][0]
    body_cell = t["content"][1]["content"][0]
    para = body_cell["content"][0]
    assert para["content"][0]["text"] == "bold"
    assert para["content"][0]["marks"] == [{"type": "strong"}]


def test_table_pipe_escape():
    md = "| h |\n| --- |\n| a\\|b |"
    doc = parse_markdown(md)
    t = doc["content"][0]
    body_cell = t["content"][1]["content"][0]
    para = body_cell["content"][0]
    assert para["content"][0]["text"] == "a|b"


def test_paragraph_then_heading():
    doc = parse_markdown("hello\n# title")
    types = [n["type"] for n in doc["content"]]
    assert types == ["paragraph", "heading"]


def test_blank_lines_between():
    doc = parse_markdown("a\n\n\n\nb")
    types = [n["type"] for n in doc["content"]]
    assert types == ["paragraph", "paragraph"]


def test_doc_envelope_attrs():
    doc = parse_markdown("hi")
    assert doc["type"] == "doc"
    assert doc["attrs"]["table_of_contents"] == {
        "enabled": False,
        "allowedLevels": [1, 2, 3],
    }
