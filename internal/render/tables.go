package render

import (
	"fmt"
	"strings"

	"github.com/nanakanok/boxnote2md-cli/internal/boxnote"
)

func renderTable(n *boxnote.Node, ctx *Context) string {
	var rows []*boxnote.Node
	for _, c := range n.Content {
		if c.Type == "table_row" {
			rows = append(rows, c)
		}
	}
	if len(rows) == 0 {
		return ""
	}

	grid := make([][]string, 0, len(rows))
	for _, row := range rows {
		var cells []string
		for _, cell := range row.Content {
			if cell.Type != "table_cell" {
				continue
			}
			cs := cell.AttrInt("colspan", 1)
			rs := cell.AttrInt("rowspan", 1)
			if cs > 1 || rs > 1 {
				ctx.Warn(fmt.Sprintf(
					"table cell has colspan=%d rowspan=%d — rendered as 1x1 (Markdown does not support span)",
					cs, rs))
			}
			cells = append(cells, renderCell(cell, ctx))
		}
		grid = append(grid, cells)
	}

	colCount := 0
	for _, r := range grid {
		if len(r) > colCount {
			colCount = len(r)
		}
	}
	for i, r := range grid {
		for len(r) < colCount {
			r = append(r, "")
		}
		grid[i] = r
	}

	var sb strings.Builder
	header := grid[0]
	sb.WriteString("| " + strings.Join(header, " | ") + " |\n")
	sep := make([]string, colCount)
	for i := range sep {
		sep[i] = "---"
	}
	sb.WriteString("| " + strings.Join(sep, " | ") + " |\n")
	for _, r := range grid[1:] {
		sb.WriteString("| " + strings.Join(r, " | ") + " |\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func renderCell(cell *boxnote.Node, ctx *Context) string {
	prev := ctx.inTableCell
	ctx.inTableCell = true
	defer func() { ctx.inTableCell = prev }()

	var parts []string
	for _, child := range cell.Content {
		var s string
		if child.Type == "paragraph" {
			s = renderChildrenInline(child, ctx)
		} else {
			s = renderBlock(child, ctx)
		}
		if s != "" {
			parts = append(parts, s)
		}
	}
	body := strings.Join(parts, "<br><br>")

	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\n", "<br>")
	body = strings.ReplaceAll(body, "|", `\|`)
	return body
}
