from __future__ import annotations

import json
from pathlib import Path
from typing import Any


class BoxNoteParseError(Exception):
    pass


def read_boxnote(path: Path) -> dict[str, Any]:
    """`.boxnote` を読み込み、ProseMirror の `doc` ノードを返す。

    新フォーマット (schema_version=1) のみサポート。トップレベルに `doc` が無ければ
    BoxNoteParseError を投げる (旧 Etherpad atext 形式は未サポート)。
    """
    try:
        raw = path.read_text(encoding="utf-8")
    except OSError as e:
        raise BoxNoteParseError(f"failed to read {path}: {e}") from e

    try:
        data = json.loads(raw)
    except json.JSONDecodeError as e:
        raise BoxNoteParseError(f"invalid JSON in {path}: {e}") from e

    if not isinstance(data, dict):
        raise BoxNoteParseError(f"top-level is not an object: {path}")

    doc = data.get("doc")
    if not isinstance(doc, dict) or doc.get("type") != "doc":
        # 旧フォーマット検出: atext + pool が並ぶ
        if "atext" in data and "pool" in data:
            raise BoxNoteParseError(
                f"{path}: legacy Etherpad atext format is not supported "
                "(only ProseMirror schema_version=1 is supported)"
            )
        raise BoxNoteParseError(f"{path}: missing 'doc' node")

    return doc
