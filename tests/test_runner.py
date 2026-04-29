from __future__ import annotations

import json
from pathlib import Path

from boxnote2md.runner import RunOptions, run

MIN_DOC = {
    "version": 1460,
    "schema_version": 1,
    "doc": {
        "type": "doc",
        "content": [
            {"type": "paragraph", "content": [{"type": "text", "text": "hello"}]}
        ],
    },
}


def _write(path: Path, doc=MIN_DOC) -> Path:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(doc), encoding="utf-8")
    return path


def test_single_file_input(tmp_path: Path):
    src = _write(tmp_path / "src" / "note.boxnote")
    out = tmp_path / "out"
    s = run(RunOptions(input_path=src, out_dir=out, image_mode="url"))
    assert s.success == 1 and s.failed == 0
    assert (out / "note.md").read_text(encoding="utf-8").strip() == "hello"


def test_directory_input_recursive(tmp_path: Path):
    _write(tmp_path / "src" / "a.boxnote")
    _write(tmp_path / "src" / "sub" / "b.boxnote")
    out = tmp_path / "out"
    s = run(RunOptions(input_path=tmp_path / "src", out_dir=out, image_mode="url"))
    assert s.success == 2
    assert (out / "a.md").exists()
    assert (out / "sub" / "b.md").exists()


def test_directory_no_recursive(tmp_path: Path):
    _write(tmp_path / "src" / "a.boxnote")
    _write(tmp_path / "src" / "sub" / "b.boxnote")
    out = tmp_path / "out"
    s = run(
        RunOptions(input_path=tmp_path / "src", out_dir=out, recursive=False, image_mode="url")
    )
    assert s.success == 1
    assert (out / "a.md").exists()
    assert not (out / "sub" / "b.md").exists()


def test_flat_collision_renames(tmp_path: Path):
    _write(tmp_path / "src" / "x.boxnote")
    _write(tmp_path / "src" / "sub" / "x.boxnote")
    out = tmp_path / "out"
    s = run(
        RunOptions(input_path=tmp_path / "src", out_dir=out, flat=True, image_mode="url")
    )
    assert s.success == 2
    assert (out / "x.md").exists()
    assert (out / "x-1.md").exists()


def test_skip_existing_unless_overwrite(tmp_path: Path):
    src = _write(tmp_path / "src" / "n.boxnote")
    out = tmp_path / "out"
    out.mkdir()
    (out / "n.md").write_text("OLD", encoding="utf-8")
    s = run(RunOptions(input_path=src, out_dir=out, image_mode="url"))
    assert s.skipped == 1
    assert (out / "n.md").read_text(encoding="utf-8") == "OLD"

    s2 = run(RunOptions(input_path=src, out_dir=out, image_mode="url", overwrite=True))
    assert s2.success == 1
    assert (out / "n.md").read_text(encoding="utf-8").strip() == "hello"


def test_failure_continues_others(tmp_path: Path):
    _write(tmp_path / "src" / "ok.boxnote")
    bad = tmp_path / "src" / "bad.boxnote"
    bad.write_text("not json", encoding="utf-8")
    out = tmp_path / "out"
    s = run(RunOptions(input_path=tmp_path / "src", out_dir=out, image_mode="url"))
    assert s.success == 1
    assert s.failed == 1
    assert s.failures[0][0] == bad


def test_dry_run_does_not_write(tmp_path: Path):
    src = _write(tmp_path / "src" / "n.boxnote")
    out = tmp_path / "out"
    s = run(RunOptions(input_path=src, out_dir=out, image_mode="url", dry_run=True))
    assert s.success == 1
    assert not (out / "n.md").exists()


def test_input_not_found(tmp_path: Path):
    s = run(RunOptions(input_path=tmp_path / "nope", out_dir=tmp_path / "out"))
    assert s.failed == 1
