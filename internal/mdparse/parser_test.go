package mdparse

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nanakanok/boxnote2md-cli/internal/boxnote"
)

func TestParseParagraph(t *testing.T) {
	doc := Parse("hello world")
	mustBlocks(t, doc, "paragraph")
	p := doc.Content[0]
	if len(p.Content) != 1 || p.Content[0].Type != "text" || p.Content[0].Text != "hello world" {
		t.Errorf("unexpected: %+v", p.Content)
	}
}

func TestParseHeading(t *testing.T) {
	doc := Parse("# H1\n\n### H3")
	mustBlocks(t, doc, "heading", "heading")
	if doc.Content[0].AttrInt("level", 0) != 1 {
		t.Error("H1 level")
	}
	if doc.Content[1].AttrInt("level", 0) != 3 {
		t.Error("H3 level")
	}
}

func TestParseHorizontalRule(t *testing.T) {
	doc := Parse("a\n\n---\n\nb")
	mustBlocks(t, doc, "paragraph", "horizontal_rule", "paragraph")
}

func TestParseBlockquote(t *testing.T) {
	doc := Parse("> first\n>\n> second")
	mustBlocks(t, doc, "blockquote")
	bq := doc.Content[0]
	if len(bq.Content) != 2 {
		t.Errorf("blockquote should have 2 paragraphs, got %d", len(bq.Content))
	}
}

func TestParseCodeBlock(t *testing.T) {
	doc := Parse("```python\nimport json\nprint(\"hi\")\n```")
	mustBlocks(t, doc, "code_block")
	cb := doc.Content[0]
	if cb.AttrString("language") != "python" {
		t.Error("language attr")
	}
	if len(cb.Content) != 1 || cb.Content[0].Text != "import json\nprint(\"hi\")" {
		t.Errorf("code text mismatch: %+v", cb.Content)
	}
}

func TestParseBulletList(t *testing.T) {
	doc := Parse("- a\n- b")
	mustBlocks(t, doc, "bullet_list")
	bl := doc.Content[0]
	if len(bl.Content) != 2 {
		t.Errorf("expected 2 items, got %d", len(bl.Content))
	}
}

func TestParseOrderedListWithStart(t *testing.T) {
	doc := Parse("3. x\n4. y")
	mustBlocks(t, doc, "ordered_list")
	ol := doc.Content[0]
	if ol.AttrInt("order", 0) != 3 {
		t.Error("order should be 3")
	}
}

func TestParseCheckList(t *testing.T) {
	doc := Parse("- [x] done\n- [ ] todo")
	mustBlocks(t, doc, "check_list")
	cl := doc.Content[0]
	if !cl.Content[0].AttrBool("checked", false) {
		t.Error("first item should be checked")
	}
	if cl.Content[1].AttrBool("checked", true) {
		t.Error("second item should be unchecked")
	}
}

func TestParseNestedBulletList(t *testing.T) {
	md := "- outer\n\n    - inner1\n    - inner2\n- outer2"
	doc := Parse(md)
	bl := doc.Content[0]
	if bl.Type != "bullet_list" {
		t.Fatalf("expected bullet_list, got %q", bl.Type)
	}
	if len(bl.Content) != 2 {
		t.Fatalf("expected 2 items, got %d", len(bl.Content))
	}
	first := bl.Content[0]
	hasNested := false
	for _, c := range first.Content {
		if c.Type == "bullet_list" {
			hasNested = true
		}
	}
	if !hasNested {
		t.Error("first item should contain nested bullet_list")
	}
}

func TestParseTableSimple(t *testing.T) {
	md := "| h1 | h2 |\n| --- | --- |\n| a | b |"
	doc := Parse(md)
	mustBlocks(t, doc, "table")
	tb := doc.Content[0]
	if len(tb.Content) != 2 {
		t.Errorf("expected 2 rows (header + 1), got %d", len(tb.Content))
	}
}

func TestInlineStrong(t *testing.T) {
	got := Parse("**bold**").Content[0].Content[0]
	if got.Text != "bold" || !hasMarkType(got.Marks, "strong") {
		t.Errorf("strong not parsed: %+v", got)
	}
}

func TestInlineEmStrongCombo(t *testing.T) {
	got := Parse("***x***").Content[0].Content[0]
	if got.Text != "x" {
		t.Errorf("text mismatch: %q", got.Text)
	}
	if !hasMarkType(got.Marks, "strong") || !hasMarkType(got.Marks, "em") {
		t.Errorf("expected strong+em, got %+v", got.Marks)
	}
}

func TestInlineLink(t *testing.T) {
	got := Parse("[label](https://e.com)").Content[0].Content[0]
	if got.Text != "label" {
		t.Error("link label")
	}
	for _, m := range got.Marks {
		if m.Type == "link" {
			var attrs map[string]string
			_ = json.Unmarshal(m.Attrs, &attrs)
			if attrs["href"] != "https://e.com" {
				t.Errorf("href mismatch: %v", attrs)
			}
			return
		}
	}
	t.Error("link mark not found")
}

func TestInlineLinkWithStrong(t *testing.T) {
	got := Parse("[**x**](https://e.com)").Content[0].Content[0]
	if got.Text != "x" {
		t.Error("inner text")
	}
	if !hasMarkType(got.Marks, "strong") || !hasMarkType(got.Marks, "link") {
		t.Errorf("expected strong+link: %+v", got.Marks)
	}
}

func TestInlineImage(t *testing.T) {
	got := Parse("![alt](https://e.com/x.png)").Content[0].Content[0]
	if got.Type != "image" {
		t.Fatalf("expected image, got %q", got.Type)
	}
	if got.AttrString("src") != "https://e.com/x.png" {
		t.Error("src")
	}
	if got.AttrString("fileName") != "x.png" {
		t.Error("fileName")
	}
}

func TestInlineUnderline(t *testing.T) {
	got := Parse("<u>x</u>").Content[0].Content[0]
	if got.Text != "x" || !hasMarkType(got.Marks, "underline") {
		t.Errorf("underline: %+v", got)
	}
}

func TestInlineHardBreak(t *testing.T) {
	doc := Parse("a  \nb")
	p := doc.Content[0]
	hasBreak := false
	for _, c := range p.Content {
		if c.Type == "hard_break" {
			hasBreak = true
		}
	}
	if !hasBreak {
		t.Errorf("expected hard_break: %+v", p.Content)
	}
}

func TestInlineUnmatchedMarkerLiteral(t *testing.T) {
	got := Parse("a**b").Content[0]
	if len(got.Content) != 1 || got.Content[0].Text != "a**b" {
		t.Errorf("unmatched ** should be literal: %+v", got.Content)
	}
}

func TestInlineEscapeBackslash(t *testing.T) {
	got := Parse(`\*not em\*`).Content[0]
	if got.Content[0].Text != "*not em*" {
		t.Errorf("escape: %q", got.Content[0].Text)
	}
}

func TestParseDocAttrs(t *testing.T) {
	doc := Parse("hi")
	if doc.Type != "doc" {
		t.Error("doc type")
	}
	var attrs map[string]any
	if err := json.Unmarshal(doc.Attrs, &attrs); err != nil {
		t.Fatal(err)
	}
	toc, ok := attrs["table_of_contents"].(map[string]any)
	if !ok {
		t.Fatal("table_of_contents missing")
	}
	if toc["enabled"].(bool) != false {
		t.Error("toc.enabled should be false")
	}
}

// ============================================================
// helpers
// ============================================================

func mustBlocks(t *testing.T, doc *boxnote.Node, types ...string) {
	t.Helper()
	if len(doc.Content) != len(types) {
		t.Fatalf("got %d blocks, want %d (types: %s)", len(doc.Content), len(types), strings.Join(blockTypes(doc.Content), ","))
	}
	for i, want := range types {
		if doc.Content[i].Type != want {
			t.Errorf("block %d type = %q, want %q", i, doc.Content[i].Type, want)
		}
	}
}

func blockTypes(blocks []*boxnote.Node) []string {
	out := make([]string, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, b.Type)
	}
	return out
}

func hasMarkType(marks []*boxnote.Mark, t string) bool {
	for _, m := range marks {
		if m.Type == t {
			return true
		}
	}
	return false
}
