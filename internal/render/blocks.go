package render

import (
	"fmt"
	"strings"

	"github.com/nanakanok/boxnote2md-cli/internal/boxnote"
)

func renderParagraph(n *boxnote.Node, ctx *Context) string {
	return renderChildrenInline(n, ctx)
}

func renderHeading(n *boxnote.Node, ctx *Context) string {
	level := n.AttrInt("level", 1)
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	inner := strings.TrimSpace(renderChildrenInline(n, ctx))
	return strings.Repeat("#", level) + " " + inner
}

func renderBlockquote(n *boxnote.Node, ctx *Context) string {
	blocks := renderChildrenBlocks(n, ctx)
	body := strings.Join(blocks, "\n\n")
	return prefixLines(body, "> ", ">")
}

func renderCodeBlock(n *boxnote.Node, ctx *Context) string {
	language := strings.ToLower(strings.TrimSpace(n.AttrString("language")))
	var sb strings.Builder
	for _, c := range n.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	return fmt.Sprintf("```%s\n%s\n```", language, sb.String())
}

func renderCallOutBox(n *boxnote.Node, ctx *Context) string {
	emoji := n.AttrString("emoji")
	bg := n.AttrString("backgroundColor")
	blocks := renderChildrenBlocks(n, ctx)
	if len(blocks) == 0 {
		return ""
	}
	if emoji != "" {
		blocks[0] = emoji + " " + blocks[0]
	}
	if ctx.KeepStyles && bg != "" {
		body := strings.Join(blocks, "\n\n")
		return fmt.Sprintf(`<div style="background-color:%s">

%s

</div>`, bg, body)
	}
	body := strings.Join(blocks, "\n\n")
	return prefixLines(body, "> ", ">")
}

// prefixLines は文字列の各行に prefix (空行には emptyPrefix) を付与する。
func prefixLines(s, prefix, emptyPrefix string) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		if ln == "" {
			lines[i] = emptyPrefix
		} else {
			lines[i] = prefix + ln
		}
	}
	return strings.Join(lines, "\n")
}
