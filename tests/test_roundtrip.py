"""ラウンドトリップ: .boxnote → md → .boxnote' の意味的等価性テスト。

完全な情報復元はできない (call_out_box, font_*, alignment 等は失われる) ので、
失われない構造に絞って比較する。
"""

from __future__ import annotations

import json
from pathlib import Path

from boxnote2md.md import parse_markdown
from boxnote2md.reader import read_boxnote
from boxnote2md.render import RenderContext, render_document

SAMPLE_PATH = Path(__file__).parent / "sample.boxnote"


def _strip(node):
    """比較しやすいように、構造の本質部分だけ残す。

    - text の `marks` のうち author_id / annotation_id を除去
    - heading の attrs.guid を除去 (生成側は付けない)
    - image の attrs から認証関連を除去 (boxSharedLink が src に来る前提)
    - call_out_box / box_preview は構造復元できないので呼び出し側でケア
    """
    if isinstance(node, dict):
        out = {}
        for k, v in node.items():
            if k == "marks":
                v = [
                    m for m in v
                    if m.get("type") not in ("author_id", "annotation_id")
                ]
                if not v:
                    continue
            elif k == "attrs":
                v = {ak: av for ak, av in v.items() if ak not in ("guid",)}
                if not v:
                    continue
            out[k] = _strip(v)
        return out
    if isinstance(node, list):
        return [_strip(x) for x in node]
    return node


def _types(node, depth=0, acc=None):
    if acc is None:
        acc = []
    if isinstance(node, dict):
        t = node.get("type")
        if t:
            acc.append(t)
        for k, v in node.items():
            if k == "content":
                _types(v, depth + 1, acc)
    elif isinstance(node, list):
        for x in node:
            _types(x, depth, acc)
    return acc


def test_sample_roundtrip_basic_blocks_preserved():
    """サンプルを md 経由で書き戻し、消えると分かっている要素を除いて構造が一致。"""
    if not SAMPLE_PATH.exists():
        return  # サンプルが無い環境ではスキップ

    doc1 = read_boxnote(SAMPLE_PATH)
    md = render_document(doc1, RenderContext(image_mode="url"))
    doc2 = parse_markdown(md)

    # 消えると分かっている type を除外して typesのリストを比較
    LOSSY_BLOCK = {"call_out_box", "box_preview"}

    def types_filtered(doc):
        out = []
        def walk(n):
            if isinstance(n, dict):
                t = n.get("type")
                if t and t not in LOSSY_BLOCK:
                    out.append(t)
                if "content" in n:
                    for c in n["content"]:
                        walk(c)
            elif isinstance(n, list):
                for x in n:
                    walk(x)
        walk(doc)
        return out

    t1 = types_filtered(doc1)
    t2 = types_filtered(doc2)

    # 厳格に同数を要求する主要構造ブロック
    strict_blocks = {
        "heading",
        "horizontal_rule",
        "bullet_list",
        "ordered_list",
        "check_list",
        "list_item",
        "check_list_item",
        "code_block",
        "table",
        "table_row",
        "image",
    }
    for b in strict_blocks:
        assert t2.count(b) == t1.count(b), (
            f"{b} count: orig={t1.count(b)}, restored={t2.count(b)}"
        )

    # 緩い期待: blockquote は call_out_box の退化分で増えうる
    assert t2.count("blockquote") >= t1.count("blockquote")

    # paragraph / table_cell は空段落の扱いやセル内段落で変動するので、
    # 「元の半分以上は残っている」ことだけ確認 (情報損失を検出するための下限)
    for b in ("paragraph", "table_cell"):
        assert t2.count(b) >= t1.count(b) // 2, (
            f"{b} severely lost: orig={t1.count(b)}, restored={t2.count(b)}"
        )


def test_minimal_md_to_boxnote(tmp_path: Path):
    md = "\n".join([
        "# Title",
        "",
        "**bold** and *italic*.",
        "",
        "- a",
        "- b",
        "",
        "1. one",
        "2. two",
        "",
        "- [x] done",
        "- [ ] todo",
        "",
        "> quote",
        "",
        "```python",
        "x = 1",
        "```",
        "",
        "| h1 | h2 |",
        "| --- | --- |",
        "| a | b |",
        "",
        "![img](https://e.com/x.png)",
    ])
    doc = parse_markdown(md)
    types = []
    def walk(n):
        if isinstance(n, dict):
            t = n.get("type")
            if t:
                types.append(t)
            if "content" in n:
                for c in n["content"]:
                    walk(c)
    walk(doc)
    expected_blocks = {
        "heading": 1,
        "paragraph": 4,  # 'bold and italic.', '![img]', and 2 inside table cells (header)
        "bullet_list": 1,
        "ordered_list": 1,
        "check_list": 1,
        "blockquote": 1,
        "code_block": 1,
        "table": 1,
        "image": 1,
    }
    for b, expected_min in expected_blocks.items():
        actual = types.count(b)
        assert actual >= expected_min, f"{b}: got {actual}, expected >= {expected_min}"


def test_md_to_boxnote_envelope_format(tmp_path: Path):
    """生成された .boxnote の envelope が必須キーを持つことを確認。"""
    from boxnote2md.md import wrap_envelope
    doc = parse_markdown("hello")
    env = wrap_envelope(doc, timestamp_ms=12345)
    assert env["version"] == 1
    assert env["schema_version"] == 1
    assert env["doc"]["type"] == "doc"
    assert env["last_edit_timestamp"] == 12345
    assert "savepoint_metadata" in env


def test_md2boxnote_runner_e2e(tmp_path: Path):
    from boxnote2md.md2boxnote_runner import Md2BoxOptions, run

    src = tmp_path / "src" / "note.md"
    src.parent.mkdir()
    src.write_text("# Hello\n\nworld", encoding="utf-8")
    out = tmp_path / "out"
    s = run(Md2BoxOptions(input_path=src, out_dir=out))
    assert s.success == 1 and s.failed == 0
    written = out / "note.boxnote"
    assert written.exists()
    env = json.loads(written.read_text(encoding="utf-8"))
    assert env["doc"]["type"] == "doc"
    assert env["doc"]["content"][0]["type"] == "heading"
