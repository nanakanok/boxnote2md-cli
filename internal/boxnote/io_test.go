package boxnote

import (
	"path/filepath"
	"testing"
)

func TestReadSampleBoxnote(t *testing.T) {
	// testdata/sample.boxnote はリポジトリルートにあるので、テスト実行時は
	// internal/boxnote/ から相対パスで参照する必要がある。
	path := filepath.Join("..", "..", "testdata", "sample.boxnote")
	doc, err := Read(path)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if doc.Type != "doc" {
		t.Fatalf("expected doc, got %q", doc.Type)
	}
	if len(doc.Content) == 0 {
		t.Fatalf("doc.content empty")
	}
}

func TestReadLegacyFormatRejected(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "legacy.boxnote")
	// 旧形式: atext + pool
	legacy := `{"atext":{"text":"hi","attribs":""},"pool":{}}`
	if err := writeString(path, legacy); err != nil {
		t.Fatal(err)
	}
	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error for legacy format")
	}
}

func TestWrapEnvelope(t *testing.T) {
	doc := &Node{Type: "doc", Content: []*Node{{Type: "paragraph"}}}
	env := WrapEnvelope(doc, 12345)
	if env.Version != 1 {
		t.Errorf("Version = %d, want 1", env.Version)
	}
	if env.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", env.SchemaVersion)
	}
	if env.LastEditTimestamp != 12345 {
		t.Errorf("LastEditTimestamp = %d, want 12345", env.LastEditTimestamp)
	}
	if env.Doc != doc {
		t.Error("Doc pointer not preserved")
	}
}

func TestWrapEnvelopeNowTimestamp(t *testing.T) {
	env := WrapEnvelope(&Node{Type: "doc"}, 0)
	if env.LastEditTimestamp == 0 {
		t.Error("LastEditTimestamp should be auto-filled when 0 passed")
	}
}

func writeString(path, s string) error {
	return writeFile(path, []byte(s))
}
