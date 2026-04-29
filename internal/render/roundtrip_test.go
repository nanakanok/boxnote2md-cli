package render

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/nanakanok/boxnote2md-cli/internal/boxnote"
	"github.com/nanakanok/boxnote2md-cli/internal/mdparse"
)

// TestRoundtripSampleBoxnote: sample.boxnote → md → boxnote' → md'
// 各段で「失われない構造」が同数であることを確認。
func TestRoundtripSampleBoxnote(t *testing.T) {
	doc1, err := boxnote.Read(filepath.Join("..", "..", "testdata", "sample.boxnote"))
	if err != nil {
		t.Fatal(err)
	}

	md1, err := Document(doc1, &Context{ImageMode: "url"})
	if err != nil {
		t.Fatal(err)
	}

	doc2 := mdparse.Parse(md1)

	// 厳密に同数を要求する主要構造ブロック
	strict := []string{
		"heading", "horizontal_rule",
		"bullet_list", "ordered_list", "check_list",
		"list_item", "check_list_item",
		"code_block",
		"table", "table_row",
		"image",
	}
	c1 := countTypes(doc1)
	c2 := countTypes(doc2)
	for _, k := range strict {
		if c1[k] != c2[k] {
			t.Errorf("type %q count: doc1=%d, doc2=%d", k, c1[k], c2[k])
		}
	}

	// blockquote: call_out_box が blockquote に退化するので doc2 >= doc1
	if c2["blockquote"] < c1["blockquote"] {
		t.Errorf("blockquote: doc1=%d, doc2=%d (should not decrease)", c1["blockquote"], c2["blockquote"])
	}

	// paragraph / table_cell は緩く: 元の半分以上残れば OK
	for _, k := range []string{"paragraph", "table_cell"} {
		if c2[k] < c1[k]/2 {
			t.Errorf("type %q severely lost: doc1=%d, doc2=%d", k, c1[k], c2[k])
		}
	}
}

// TestRoundtripStability: doc1 → md1 → doc2 → md2 で md1 == md2 (idempotent stage).
func TestRoundtripStability(t *testing.T) {
	doc1, err := boxnote.Read(filepath.Join("..", "..", "testdata", "sample.boxnote"))
	if err != nil {
		t.Fatal(err)
	}
	md1, err := Document(doc1, &Context{ImageMode: "url"})
	if err != nil {
		t.Fatal(err)
	}
	doc2 := mdparse.Parse(md1)
	md2, err := Document(doc2, &Context{ImageMode: "url"})
	if err != nil {
		t.Fatal(err)
	}
	// md1 と md2 は完全一致しないかもしれないが、行数が大幅にずれてはいけない
	l1 := len(strings.Split(md1, "\n"))
	l2 := len(strings.Split(md2, "\n"))
	if l1 == 0 || l2 == 0 {
		t.Fatalf("empty output: l1=%d l2=%d", l1, l2)
	}
	ratio := float64(l2) / float64(l1)
	if ratio < 0.9 || ratio > 1.1 {
		t.Errorf("md line count drift: l1=%d l2=%d (ratio %.2f)", l1, l2, ratio)
	}
}

func countTypes(node *boxnote.Node) map[string]int {
	c := map[string]int{}
	var walk func(*boxnote.Node)
	walk = func(n *boxnote.Node) {
		if n == nil {
			return
		}
		c[n.Type]++
		for _, ch := range n.Content {
			walk(ch)
		}
	}
	walk(node)
	return c
}
