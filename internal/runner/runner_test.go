package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const minDoc = `{"version":1460,"schema_version":1,"doc":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hello"}]}]}}`

func writeBoxnote(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunBoxnoteToMdSingleFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src", "note.boxnote")
	writeBoxnote(t, src, minDoc)
	out := filepath.Join(tmp, "out")

	s, err := RunBoxnoteToMd(BoxnoteToMdOptions{
		Input:     src,
		OutDir:    out,
		ImageMode: "url",
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Success != 1 || s.Failed != 0 {
		t.Errorf("summary: %+v", s)
	}
	got, err := os.ReadFile(filepath.Join(out, "note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello\n" {
		t.Errorf("got %q, want %q", string(got), "hello\n")
	}
}

func TestRunBoxnoteToMdDirectoryRecursive(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	writeBoxnote(t, filepath.Join(srcDir, "a.boxnote"), minDoc)
	writeBoxnote(t, filepath.Join(srcDir, "sub", "b.boxnote"), minDoc)
	out := filepath.Join(tmp, "out")

	s, err := RunBoxnoteToMd(BoxnoteToMdOptions{
		Input:     srcDir,
		OutDir:    out,
		Recursive: true,
		ImageMode: "url",
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Success != 2 {
		t.Errorf("expected 2 success, got %d", s.Success)
	}
	if _, err := os.Stat(filepath.Join(out, "a.md")); err != nil {
		t.Errorf("a.md missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "sub", "b.md")); err != nil {
		t.Errorf("sub/b.md missing: %v", err)
	}
}

func TestRunBoxnoteToMdNoRecursive(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	writeBoxnote(t, filepath.Join(srcDir, "a.boxnote"), minDoc)
	writeBoxnote(t, filepath.Join(srcDir, "sub", "b.boxnote"), minDoc)
	out := filepath.Join(tmp, "out")

	s, err := RunBoxnoteToMd(BoxnoteToMdOptions{
		Input:     srcDir,
		OutDir:    out,
		Recursive: false,
		ImageMode: "url",
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Success != 1 {
		t.Errorf("expected 1 success, got %d", s.Success)
	}
}

func TestRunBoxnoteToMdSkipExisting(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src", "n.boxnote")
	writeBoxnote(t, src, minDoc)
	out := filepath.Join(tmp, "out")
	_ = os.MkdirAll(out, 0o755)
	_ = os.WriteFile(filepath.Join(out, "n.md"), []byte("OLD"), 0o644)

	s1, _ := RunBoxnoteToMd(BoxnoteToMdOptions{Input: src, OutDir: out, ImageMode: "url"})
	if s1.Skipped != 1 {
		t.Errorf("first call should skip, got %+v", s1)
	}
	got, _ := os.ReadFile(filepath.Join(out, "n.md"))
	if string(got) != "OLD" {
		t.Errorf("file overwritten while not asked: %q", got)
	}

	s2, _ := RunBoxnoteToMd(BoxnoteToMdOptions{Input: src, OutDir: out, ImageMode: "url", Overwrite: true})
	if s2.Success != 1 {
		t.Errorf("with Overwrite: %+v", s2)
	}
}

func TestRunBoxnoteToMdFailureContinues(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	writeBoxnote(t, filepath.Join(srcDir, "ok.boxnote"), minDoc)
	bad := filepath.Join(srcDir, "bad.boxnote")
	_ = os.WriteFile(bad, []byte("not json"), 0o644)
	out := filepath.Join(tmp, "out")

	s, _ := RunBoxnoteToMd(BoxnoteToMdOptions{
		Input:     srcDir,
		OutDir:    out,
		Recursive: true,
		ImageMode: "url",
	})
	if s.Success != 1 || s.Failed != 1 {
		t.Errorf("got %+v", s)
	}
}

func TestRunBoxnoteToMdDryRun(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "n.boxnote")
	writeBoxnote(t, src, minDoc)
	out := filepath.Join(tmp, "out")

	s, _ := RunBoxnoteToMd(BoxnoteToMdOptions{
		Input:     src,
		OutDir:    out,
		ImageMode: "url",
		DryRun:    true,
	})
	if s.Success != 1 {
		t.Errorf("dry-run should still count success: %+v", s)
	}
	if _, err := os.Stat(filepath.Join(out, "n.md")); err == nil {
		t.Error("dry-run should not write file")
	}
}

func TestRunMdToBoxnote(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src", "note.md")
	_ = os.MkdirAll(filepath.Dir(src), 0o755)
	_ = os.WriteFile(src, []byte("# Hello\n\nworld"), 0o644)
	out := filepath.Join(tmp, "out")

	s, err := RunMdToBoxnote(MdToBoxnoteOptions{
		Input:  src,
		OutDir: out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Success != 1 {
		t.Errorf("got %+v", s)
	}
	dest := filepath.Join(out, "note.boxnote")
	raw, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	var env map[string]any
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if env["schema_version"].(float64) != 1 {
		t.Error("schema_version")
	}
	doc := env["doc"].(map[string]any)
	if doc["type"] != "doc" {
		t.Error("doc.type")
	}
}

func TestFlatCollisionRenames(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	writeBoxnote(t, filepath.Join(srcDir, "x.boxnote"), minDoc)
	writeBoxnote(t, filepath.Join(srcDir, "sub", "x.boxnote"), minDoc)
	out := filepath.Join(tmp, "out")

	s, _ := RunBoxnoteToMd(BoxnoteToMdOptions{
		Input:     srcDir,
		OutDir:    out,
		Recursive: true,
		Flat:      true,
		ImageMode: "url",
	})
	if s.Success != 2 {
		t.Errorf("got %+v", s)
	}
	if _, err := os.Stat(filepath.Join(out, "x.md")); err != nil {
		t.Error("x.md missing")
	}
	if _, err := os.Stat(filepath.Join(out, "x-1.md")); err != nil {
		t.Error("x-1.md missing")
	}
}
