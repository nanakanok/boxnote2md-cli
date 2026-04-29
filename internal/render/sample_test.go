package render

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/nanakanok/boxnote2md-cli/internal/boxnote"
)

func TestRenderSampleBoxnote(t *testing.T) {
	doc, err := boxnote.Read(filepath.Join("..", "..", "testdata", "sample.boxnote"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	ctx := &Context{ImageMode: "url"}
	md, err := Document(doc, ctx)
	if err != nil {
		t.Fatalf("Document: %v", err)
	}
	// 主要要素が含まれていることを確認
	for _, want := range []string{
		"**Text With Bold**",
		"*Italic Text*",
		"<u>Underline text</u>",
		"~~Delete Text~~",
		"***Bold and Italic***",
		"# Heading 1",
		"## Heading 2",
		"### Heading 3",
		"---",
		"```python\nimport json",
		"> ⚠️ Callout!",
		"> BlockQuote",
		"| Table (1, 1) |",
		"![Drawing1.png](https://mover.box.com/s/k4ya8p39nw8gdk1a9wzyev17q9wlaepj)",
		"[Box: 33z7yjwptj8zv7wyu2y4qythx219mefe](https://mover.box.com",
		"- Bullet list 1",
		"- [x] CheckList 1",
		"- [ ] **CheckList 2 bold**",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("output missing expected fragment: %q", want)
		}
	}
}
