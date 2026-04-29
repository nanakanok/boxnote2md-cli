"""画像 (image) と Box プレビュー (box_preview) の Markdown 化。

- image:
    - --image-mode download: boxSharedLink を取得して image_dir に保存し、
      MD には <md_path> から見た相対パスで ![alt](path) を出力。
    - --image-mode url: boxSharedLink をそのまま ![alt](url)。
    - boxSharedLink が空: ![image:fileName](#unavailable)。
    - DL 失敗時は警告 + URL フォールバック。
- box_preview: MD で再現できないので [Box: <fileName>](<boxSharedLink>) リンクへ退避。
"""

from __future__ import annotations

import os
import re
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

from boxnote2md.render.document import Node, RenderContext

# 同一プロセス内で同じ URL を二度叩かないためのキャッシュ
_DL_CACHE: dict[str, Path | None] = {}


def _safe_filename(name: str) -> str:
    name = name.replace("/", "_").replace("\\", "_")
    name = re.sub(r"[\x00-\x1f]", "", name)
    return name or "file"


def _relative_path(target: Path, anchor: Path | None) -> str:
    """anchor (出力する .md) から target への相対パスを POSIX 形式で返す。"""
    target = target.resolve()
    if anchor is None:
        return target.as_posix()
    try:
        rel = os.path.relpath(target, anchor.resolve().parent)
    except ValueError:
        return target.as_posix()
    return Path(rel).as_posix()


def _download(url: str, dest_dir: Path, file_name: str, ctx: RenderContext) -> Path | None:
    """URL からファイルを取得し dest_dir/file_name に保存。失敗時は None。"""
    if url in _DL_CACHE:
        return _DL_CACHE[url]

    dest_dir.mkdir(parents=True, exist_ok=True)
    dest = dest_dir / file_name
    try:
        req = urllib.request.Request(url, headers={"User-Agent": "boxnote2md/0.1"})
        with urllib.request.urlopen(req, timeout=30) as resp:  # noqa: S310 (既知の信頼URL)
            data = resp.read()
        dest.write_bytes(data)
    except (urllib.error.URLError, TimeoutError, OSError) as e:
        ctx.warn(f"image download failed for {url}: {e}")
        _DL_CACHE[url] = None
        return None

    _DL_CACHE[url] = dest
    return dest


def render_image(node: Node, ctx: RenderContext) -> str:
    attrs = node.get("attrs", {}) or {}
    box_link: str = attrs.get("boxSharedLink") or ""
    box_file_id: str = attrs.get("boxFileId") or ""
    file_name: str = attrs.get("fileName") or "image"
    alt: str = attrs.get("alt") or file_name or "image"
    src: str = attrs.get("src") or ""

    # 取得元 URL 決定: src が直接URLなら優先、無ければ boxSharedLink
    target_url = src if src.startswith(("http://", "https://")) else box_link

    if not target_url:
        ctx.warn(f"image without URL (fileName={file_name!r})")
        return f"![image:{file_name}](#unavailable)"

    if ctx.image_mode == "url":
        return f"![{alt}]({target_url})"

    # download モード
    safe_name = _safe_filename(f"{box_file_id}__{file_name}" if box_file_id else file_name)
    image_dir = ctx.image_dir or Path("./out/images")
    saved = _download(target_url, image_dir, safe_name, ctx)
    if saved is None:
        # フォールバック: 元 URL を埋め込み
        return f"![{alt}]({target_url})"
    rel = _relative_path(saved, ctx.md_path)
    ctx.image_results.append({"src_url": target_url, "saved_to": str(saved)})
    return f"![{alt}]({rel})"


def render_box_preview(node: Node, _ctx: RenderContext) -> str:
    attrs = node.get("attrs", {}) or {}
    link: str = attrs.get("boxSharedLink") or ""
    file_name = attrs.get("fileName") or _basename_from_url(link) or "Box file"
    if not link:
        return f"[Box: {file_name}](#unavailable)"
    return f"[Box: {file_name}]({link})"


def _basename_from_url(url: str) -> str:
    if not url:
        return ""
    parts = urllib.parse.urlparse(url)
    return parts.path.rsplit("/", 1)[-1]


RENDERERS = {
    "image": render_image,
    "box_preview": render_box_preview,
}
