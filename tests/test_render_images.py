from __future__ import annotations

from pathlib import Path
from unittest.mock import patch

from boxnote2md.render import RenderContext, render_document
from boxnote2md.render import images as images_mod


def doc(*blocks):
    return {"type": "doc", "content": list(blocks)}


def image(**attrs):
    base = {
        "src": "",
        "alt": "",
        "boxSharedLink": "",
        "boxFileId": "",
        "fileName": "img.png",
        "width": None,
        "height": None,
    }
    base.update(attrs)
    return {"type": "image", "attrs": base}


def setup_function():
    # キャッシュをリセット
    images_mod._DL_CACHE.clear()


def test_image_url_mode():
    md = render_document(
        doc(image(boxSharedLink="https://e.com/x.png", fileName="x.png", alt="x")),
        RenderContext(image_mode="url"),
    )
    assert md == "![x](https://e.com/x.png)\n"


def test_image_url_mode_uses_filename_when_alt_empty():
    md = render_document(
        doc(image(boxSharedLink="https://e.com/x.png", fileName="diagram.png")),
        RenderContext(image_mode="url"),
    )
    assert md == "![diagram.png](https://e.com/x.png)\n"


def test_image_no_url_renders_placeholder():
    md = render_document(
        doc(image(boxSharedLink="", fileName="missing.png")),
        RenderContext(image_mode="url"),
    )
    assert md == "![image:missing.png](#unavailable)\n"


def test_image_download_mode_saves_and_relative_path(tmp_path: Path):
    # _download を差し替えてネットワークを叩かないようにする
    saved_to = tmp_path / "imgdir" / "fid__pic.png"

    def fake_dl(url, dest_dir, file_name, ctx):
        path = dest_dir / file_name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(b"x")
        return path

    md_path = tmp_path / "doc.md"
    ctx = RenderContext(
        image_mode="download",
        image_dir=tmp_path / "imgdir",
        md_path=md_path,
    )
    with patch.object(images_mod, "_download", side_effect=fake_dl):
        md = render_document(
            doc(
                image(
                    boxSharedLink="https://e.com/p.png",
                    boxFileId="fid",
                    fileName="pic.png",
                    alt="pic",
                )
            ),
            ctx,
        )
    assert md == "![pic](imgdir/fid__pic.png)\n"
    assert saved_to.read_bytes() == b"x"
    assert ctx.image_results and ctx.image_results[0]["src_url"] == "https://e.com/p.png"


def test_image_download_failure_falls_back_to_url(tmp_path: Path):
    md_path = tmp_path / "doc.md"
    ctx = RenderContext(image_mode="download", image_dir=tmp_path / "imgs", md_path=md_path)
    with patch.object(images_mod, "_download", return_value=None):
        md = render_document(
            doc(image(boxSharedLink="https://e.com/p.png", fileName="p.png")),
            ctx,
        )
    assert md == "![p.png](https://e.com/p.png)\n"


def test_box_preview_link_retreat():
    md = render_document(
        doc(
            {
                "type": "box_preview",
                "attrs": {"boxSharedLink": "https://e.com/s/abc"},
            }
        ),
        RenderContext(),
    )
    assert md.startswith("[Box: ")
    assert "(https://e.com/s/abc)" in md


def test_box_preview_no_link():
    md = render_document(
        doc({"type": "box_preview", "attrs": {}}),
        RenderContext(),
    )
    assert md == "[Box: Box file](#unavailable)\n"
