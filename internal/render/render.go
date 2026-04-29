// Package render は ProseMirror `doc` を Markdown 文字列に変換する。
package render

import (
	"fmt"
	"strings"

	"github.com/nanakanok/boxnote2md-cli/internal/boxnote"
)

// Context は変換中の状態を保持する。
type Context struct {
	KeepStyles bool
	ImageMode  string // "download" | "url"

	// ImageDir は画像保存先の絶対/相対パス (download モード時)。
	ImageDir string
	// MdPath は出力 .md のパス。画像相対参照の計算に使う。
	MdPath string

	// 結果として記録する画像ダウンロード/参照情報。
	ImageResults []ImageResult
	// 警告メッセージ。
	Warnings []string

	// 内部状態
	listDepth   int
	inTableCell bool

	// ImageDownloader は test 等から差し替え可能な画像取得関数。
	// 戻り値: 保存先パス (相対参照に使う) または取得失敗時は空文字。
	ImageDownloader func(url, destDir, fileName string, ctx *Context) string
}

// ImageResult は画像 1 件分の処理結果。
type ImageResult struct {
	SrcURL  string
	SavedTo string
}

// Warn は警告を記録する。
func (c *Context) Warn(msg string) {
	c.Warnings = append(c.Warnings, msg)
}

// Document は doc ノードを Markdown 文字列に変換する。
func Document(doc *boxnote.Node, ctx *Context) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("nil doc")
	}
	if doc.Type != "doc" {
		return "", fmt.Errorf("expected doc node, got %q", doc.Type)
	}
	if ctx == nil {
		ctx = &Context{}
	}
	var parts []string
	for _, child := range doc.Content {
		s := renderBlock(child, ctx)
		s = strings.TrimRight(s, "\n")
		if s == "" {
			continue
		}
		parts = append(parts, s)
	}
	body := strings.Join(parts, "\n\n")
	body = strings.TrimRight(body, "\n") + "\n"
	return body, nil
}

// renderBlock はディスパッチャ。
func renderBlock(n *boxnote.Node, ctx *Context) string {
	switch n.Type {
	case "paragraph":
		return renderParagraph(n, ctx)
	case "heading":
		return renderHeading(n, ctx)
	case "horizontal_rule":
		return "---"
	case "blockquote":
		return renderBlockquote(n, ctx)
	case "code_block":
		return renderCodeBlock(n, ctx)
	case "call_out_box":
		return renderCallOutBox(n, ctx)
	case "bullet_list":
		return renderBulletList(n, ctx)
	case "ordered_list":
		return renderOrderedList(n, ctx)
	case "check_list":
		return renderCheckList(n, ctx)
	case "table":
		return renderTable(n, ctx)
	case "image":
		return renderImage(n, ctx)
	case "box_preview":
		return renderBoxPreview(n, ctx)
	case "list_item", "check_list_item", "table_row", "table_cell":
		// 親ブロックから直接呼ばれない構造内ノード
		ctx.Warn(fmt.Sprintf("unexpected top-level %s", n.Type))
		return renderChildrenInline(n, ctx)
	default:
		ctx.Warn(fmt.Sprintf("unknown block type: %q — passing through children", n.Type))
		return renderChildrenInline(n, ctx)
	}
}

// renderChildrenInline は子をインラインとして連結する。
// block / inline 混在時は block も呼ぶが、その場合改行が含まれる。
func renderChildrenInline(n *boxnote.Node, ctx *Context) string {
	var b strings.Builder
	for _, c := range n.Content {
		switch c.Type {
		case "text":
			b.WriteString(renderTextWithMarks(c, ctx))
		case "hard_break":
			if ctx.inTableCell {
				b.WriteString("<br>")
			} else {
				b.WriteString("  \n")
			}
		case "image":
			b.WriteString(renderImage(c, ctx))
		case "box_preview":
			b.WriteString(renderBoxPreview(c, ctx))
		default:
			// その他のノードはブロックレンダラに任せる
			b.WriteString(renderBlock(c, ctx))
		}
	}
	return b.String()
}

// renderChildrenBlocks はブロック子要素を順にレンダリングして文字列スライスで返す。
func renderChildrenBlocks(n *boxnote.Node, ctx *Context) []string {
	var out []string
	for _, c := range n.Content {
		var s string
		if c.Type == "paragraph" {
			s = renderChildrenInline(c, ctx)
		} else {
			s = renderBlock(c, ctx)
		}
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
