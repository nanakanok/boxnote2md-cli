package render

import (
	"fmt"
	"strings"

	"github.com/nanakanok/boxnote2md-cli/internal/boxnote"
)

const indent = "    " // GFM の継続行は 4 スペース

func renderBulletList(n *boxnote.Node, ctx *Context) string {
	ctx.listDepth++
	defer func() { ctx.listDepth-- }()

	var lines []string
	for _, item := range n.Content {
		blocks := listItemBlocks(item, ctx)
		lines = append(lines, joinListItem("- ", blocks))
	}
	return strings.Join(lines, "\n")
}

func renderOrderedList(n *boxnote.Node, ctx *Context) string {
	ctx.listDepth++
	defer func() { ctx.listDepth-- }()

	start := n.AttrInt("order", 1)
	if start < 1 {
		start = 1
	}
	var lines []string
	for i, item := range n.Content {
		blocks := listItemBlocks(item, ctx)
		prefix := fmt.Sprintf("%d. ", start+i)
		lines = append(lines, joinListItem(prefix, blocks))
	}
	return strings.Join(lines, "\n")
}

func renderCheckList(n *boxnote.Node, ctx *Context) string {
	ctx.listDepth++
	defer func() { ctx.listDepth-- }()

	var lines []string
	for _, item := range n.Content {
		mark := "[ ]"
		if item.AttrBool("checked", false) {
			mark = "[x]"
		}
		blocks := listItemBlocks(item, ctx)
		lines = append(lines, joinListItem(fmt.Sprintf("- %s ", mark), blocks))
	}
	return strings.Join(lines, "\n")
}

// listItemBlocks は list_item / check_list_item の中身を文字列リストに変換する。
func listItemBlocks(item *boxnote.Node, ctx *Context) []string {
	var out []string
	for _, child := range item.Content {
		var s string
		switch child.Type {
		case "paragraph":
			s = renderChildrenInline(child, ctx)
		case "bullet_list", "ordered_list", "check_list":
			s = renderBlock(child, ctx)
		default:
			s = renderBlock(child, ctx)
		}
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// joinListItem は先頭ブロックを prefix と連結し、残りを 4 スペースインデントで続ける。
func joinListItem(prefix string, blocks []string) string {
	if len(blocks) == 0 {
		return strings.TrimRight(prefix, " ")
	}
	var lines []string
	first := blocks[0]
	firstLines := strings.Split(first, "\n")
	lines = append(lines, prefix+firstLines[0])
	for _, ln := range firstLines[1:] {
		if ln == "" {
			lines = append(lines, "")
		} else {
			lines = append(lines, indent+ln)
		}
	}
	for _, blk := range blocks[1:] {
		lines = append(lines, "") // 空行で段落区切り
		for _, ln := range strings.Split(blk, "\n") {
			if ln == "" {
				lines = append(lines, "")
			} else {
				lines = append(lines, indent+ln)
			}
		}
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}
