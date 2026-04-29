"""ProseMirror `doc` → Box Note `.boxnote` トップレベル JSON。"""

from __future__ import annotations

import time
from typing import Any


def wrap_envelope(doc: dict[str, Any], *, timestamp_ms: int | None = None) -> dict[str, Any]:
    """`doc` ノードを Box Note のトップレベル JSON に包む。"""
    ts = timestamp_ms if timestamp_ms is not None else int(time.time() * 1000)
    return {
        "version": 1,
        "schema_version": 1,
        "doc": doc,
        "savepoint_metadata": {},
        "last_edit_timestamp": ts,
    }
