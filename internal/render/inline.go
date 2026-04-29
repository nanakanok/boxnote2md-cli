package render

import (
	"fmt"
	"strings"

	"github.com/nanakanok/boxnote2md-cli/internal/boxnote"
)

// renderTextWithMarks は text ノードに marks を適用して Markdown 文字列を返す。
// wrap 順序 (内側 → 外側): underline/strike/em/strong/link を経由し、最終的に link は最外。
func renderTextWithMarks(n *boxnote.Node, ctx *Context) string {
	if n.Text == "" {
		return ""
	}
	rendered := escapeMarkdown(n.Text)

	// マークを type → mark map で管理
	marks := make(map[string]*boxnote.Mark, len(n.Marks))
	for _, m := range n.Marks {
		marks[m.Type] = m
	}

	// presentational marks
	for _, k := range []string{"font_size", "font_color", "highlight"} {
		if !ctx.KeepStyles {
			delete(marks, k)
		}
	}

	if ctx.KeepStyles {
		var styleParts []string
		if m, ok := marks["font_size"]; ok {
			if size := m.AttrString("size"); size != "" {
				styleParts = append(styleParts, "font-size:"+size)
			}
			delete(marks, "font_size")
		}
		if m, ok := marks["font_color"]; ok {
			if color := m.AttrString("color"); color != "" {
				styleParts = append(styleParts, "color:"+color)
			}
			delete(marks, "font_color")
		}
		if m, ok := marks["highlight"]; ok {
			if color := m.AttrString("color"); color != "" {
				styleParts = append(styleParts, "background-color:"+color)
			}
			delete(marks, "highlight")
		}
		if len(styleParts) > 0 {
			rendered = fmt.Sprintf(`<span style="%s">%s</span>`, strings.Join(styleParts, ";"), rendered)
		}
	}

	// 標準マーク wrap (内側 → 外側)
	wrapOrder := []string{"strikethrough", "underline", "em", "strong"}
	for _, t := range wrapOrder {
		if _, ok := marks[t]; !ok {
			continue
		}
		left, right := wrapTokens(t)
		if rendered != "" {
			rendered = left + rendered + right
		}
	}
	// link は最外
	if m, ok := marks["link"]; ok {
		href := m.AttrString("href")
		rendered = fmt.Sprintf("[%s](%s)", rendered, href)
	}
	return rendered
}

func wrapTokens(markType string) (string, string) {
	switch markType {
	case "strong":
		return "**", "**"
	case "em":
		return "*", "*"
	case "underline":
		return "<u>", "</u>"
	case "strikethrough":
		return "~~", "~~"
	default:
		return "", ""
	}
}

// escapeMarkdown は最小限のエスケープのみ行う。
// 行頭の特殊文字 (#, -, > など) は段落整形側で扱う想定。
func escapeMarkdown(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		"`", "\\`",
	)
	return r.Replace(s)
}
