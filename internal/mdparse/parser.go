// Package mdparse は Markdown を ProseMirror `doc` ノードに変換する。
// CommonMark 厳密準拠ではなく、本ツール (boxnote2md) が出力する Markdown の
// ラウンドトリップを優先したサブセット実装。
package mdparse

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/nanakanok/boxnote2md-cli/internal/boxnote"
)

var (
	reHeading    = regexp.MustCompile(`^(#{1,6})[ \t]+(.*?)\s*#*\s*$`)
	reHR         = regexp.MustCompile(`^[ \t]{0,3}(?:-[ \t]*){3,}$|^[ \t]{0,3}(?:\*[ \t]*){3,}$|^[ \t]{0,3}(?:_[ \t]*){3,}$`)
	reFenceOpen  = regexp.MustCompile("^([ \\t]{0,3})(```+|~~~+)[ \\t]*([^\\s`]*)[ \\t]*$")
	reBlockquote = regexp.MustCompile(`^[ \t]{0,3}>[ \t]?(.*)$`)
	reBullet     = regexp.MustCompile(`^([ \t]*)([-*+])[ \t]+(.*)$`)
	reOrdered    = regexp.MustCompile(`^([ \t]*)(\d+)[.)][ \t]+(.*)$`)
	reCheckbox   = regexp.MustCompile(`^\[([ xX])\][ \t]+(.*)$`)
	reTableSep   = regexp.MustCompile(`^[ \t]*\|[ \t]*:?-{3,}:?[ \t]*(\|[ \t]*:?-{3,}:?[ \t]*)*\|[ \t]*$`)
	reTableRow   = regexp.MustCompile(`^[ \t]*\|.*\|[ \t]*$`)
)

// Parse は Markdown 文字列をパースして ProseMirror の doc ノードを返す。
func Parse(text string) *boxnote.Node {
	// BOM 剥がし (UTF-8 BOM = U+FEFF)
	text = strings.TrimPrefix(text, "\ufeff")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	blocks := parseBlocks(lines, 0, len(lines))

	tocAttrs, _ := json.Marshal(map[string]any{
		"table_of_contents": map[string]any{
			"enabled":       false,
			"allowedLevels": []int{1, 2, 3},
		},
	})
	return &boxnote.Node{
		Type:    "doc",
		Attrs:   tocAttrs,
		Content: blocks,
	}
}

// WrapEnvelope は doc ノードを Box Note の envelope に包む。
func WrapEnvelope(doc *boxnote.Node) *boxnote.Envelope {
	return boxnote.WrapEnvelope(doc, time.Now().UnixMilli())
}

// ============================================================
// ブロック層
// ============================================================

func parseBlocks(lines []string, start, end int) []*boxnote.Node {
	var out []*boxnote.Node
	i := start
	for i < end {
		line := lines[i]

		if strings.TrimSpace(line) == "" {
			i++
			continue
		}

		if m := reHeading.FindStringSubmatch(line); m != nil {
			level := len(m[1])
			content := strings.TrimSpace(m[2])
			node := &boxnote.Node{
				Type:  "heading",
				Attrs: jsonAttrs(map[string]any{"level": level}),
			}
			if content != "" {
				node.Content = parseInline(content)
			}
			out = append(out, node)
			i++
			continue
		}

		if reHR.MatchString(line) {
			out = append(out, &boxnote.Node{Type: "horizontal_rule"})
			i++
			continue
		}

		if m := reFenceOpen.FindStringSubmatch(line); m != nil {
			i = consumeCodeBlock(lines, i, end, m, &out)
			continue
		}

		if reBlockquote.MatchString(line) {
			i = consumeBlockquote(lines, i, end, &out)
			continue
		}

		if reTableRow.MatchString(line) && i+1 < end && reTableSep.MatchString(lines[i+1]) {
			node, consumed := parseTable(lines, i, end)
			out = append(out, node)
			i = consumed
			continue
		}

		if reBullet.MatchString(line) || reOrdered.MatchString(line) {
			node, consumed := parseList(lines, i, end)
			out = append(out, node)
			i = consumed
			continue
		}

		// 段落
		var paraLines []string
		for i < end {
			ln := lines[i]
			if strings.TrimSpace(ln) == "" {
				break
			}
			if reHeading.MatchString(ln) || reHR.MatchString(ln) || reFenceOpen.MatchString(ln) ||
				reBlockquote.MatchString(ln) || reBullet.MatchString(ln) || reOrdered.MatchString(ln) {
				break
			}
			if reTableRow.MatchString(ln) && i+1 < end && reTableSep.MatchString(lines[i+1]) {
				break
			}
			paraLines = append(paraLines, ln)
			i++
		}
		if len(paraLines) > 0 {
			text := strings.TrimSpace(strings.Join(paraLines, "\n"))
			if text != "" {
				out = append(out, &boxnote.Node{
					Type:    "paragraph",
					Content: parseInline(text),
				})
			}
		}
	}
	return out
}

func consumeCodeBlock(lines []string, start, end int, openMatch []string, out *[]*boxnote.Node) int {
	indentLen := len(openMatch[1])
	fence := openMatch[2]
	language := strings.TrimSpace(openMatch[3])
	i := start + 1
	var codeLines []string
	closeRe := regexp.MustCompile(`^[ \t]{0,3}` + regexp.QuoteMeta(string(fence[0])) + `{` + numStr(len(fence)) + `,}\s*$`)
	for i < end && !closeRe.MatchString(lines[i]) {
		ln := lines[i]
		if indentLen > 0 && len(ln) >= indentLen && strings.TrimSpace(ln[:indentLen]) == "" {
			ln = ln[indentLen:]
		}
		codeLines = append(codeLines, ln)
		i++
	}
	if i < end {
		i++
	}
	text := strings.Join(codeLines, "\n")
	node := &boxnote.Node{
		Type:  "code_block",
		Attrs: jsonAttrs(map[string]any{"language": language}),
	}
	if text != "" {
		node.Content = []*boxnote.Node{{Type: "text", Text: text}}
	}
	*out = append(*out, node)
	return i
}

func consumeBlockquote(lines []string, start, end int, out *[]*boxnote.Node) int {
	j := start
	var stripped []string
	for j < end {
		if m := reBlockquote.FindStringSubmatch(lines[j]); m != nil {
			stripped = append(stripped, m[1])
			j++
		} else if strings.TrimSpace(lines[j]) == "" {
			break
		} else {
			// lazy continuation
			stripped = append(stripped, lines[j])
			j++
		}
	}
	inner := parseBlocks(stripped, 0, len(stripped))
	*out = append(*out, &boxnote.Node{Type: "blockquote", Content: inner})
	return j
}

// ============================================================
// リスト
// ============================================================

type listItemRaw struct {
	indent       int
	markerWidth  int
	isOrdered    bool
	order        int
	isCheck      bool
	checked      bool
	contentLines []string
}

// classifyListMarker: 戻り値の最後の bool は「リストかどうか」。
func classifyListMarker(line string) (indent, markerWidth int, isOrdered bool, order int, rest string, ok bool) {
	if m := reBullet.FindStringSubmatch(line); m != nil {
		prefix := m[1]
		marker := m[2]
		rest = m[3]
		indent = lenExpand(prefix)
		markerPos := len(prefix)
		bodyPos := markerPos + len(marker)
		for bodyPos < len(line) && (line[bodyPos] == ' ' || line[bodyPos] == '\t') {
			bodyPos++
		}
		markerWidth = bodyPos - markerPos
		return indent, markerWidth, false, 0, rest, true
	}
	if m := reOrdered.FindStringSubmatch(line); m != nil {
		prefix := m[1]
		orderStr := m[2]
		rest = m[3]
		indent = lenExpand(prefix)
		markerPos := len(prefix)
		bodyPos := markerPos + len(orderStr) + 1
		for bodyPos < len(line) && (line[bodyPos] == ' ' || line[bodyPos] == '\t') {
			bodyPos++
		}
		markerWidth = bodyPos - markerPos
		order = atoi(orderStr)
		return indent, markerWidth, true, order, rest, true
	}
	return 0, 0, false, 0, "", false
}

func parseList(lines []string, start, end int) (*boxnote.Node, int) {
	baseIndent, _, baseOrdered, baseOrder, _, ok := classifyListMarker(lines[start])
	if !ok {
		return nil, start
	}

	var items []listItemRaw
	i := start
	for i < end {
		line := lines[i]
		indent, markerWidth, isOrdered, order, rest, ok := classifyListMarker(line)
		if !ok || indent != baseIndent || isOrdered != baseOrdered {
			break
		}

		isCheck := false
		checked := false
		if cm := reCheckbox.FindStringSubmatch(rest); cm != nil {
			isCheck = true
			checked = cm[1] == "x" || cm[1] == "X"
			rest = cm[2]
		}

		item := listItemRaw{
			indent:       indent,
			markerWidth:  markerWidth,
			isOrdered:    isOrdered,
			order:        order,
			isCheck:      isCheck,
			checked:      checked,
			contentLines: []string{rest},
		}
		i++
		contIndent := indent + markerWidth

		for i < end {
			ln := lines[i]
			if strings.TrimSpace(ln) == "" {
				if i+1 < end {
					nxt := lines[i+1]
					if nxtIndent, _, _, _, _, nxtOK := classifyListMarker(nxt); nxtOK && nxtIndent == baseIndent {
						break
					}
					nxtWS := leadingWS(nxt)
					if strings.TrimSpace(nxt) != "" && nxtWS >= contIndent {
						item.contentLines = append(item.contentLines, "")
						i++
						continue
					}
				}
				break
			}
			lnIndent, _, _, _, _, lnOK := classifyListMarker(ln)
			if lnOK && lnIndent == baseIndent {
				break
			}
			ws := leadingWS(ln)
			if ws >= contIndent || lnOK {
				if ws >= contIndent {
					item.contentLines = append(item.contentLines, ln[contIndent:])
				} else {
					item.contentLines = append(item.contentLines, ln)
				}
				i++
				continue
			}
			// lazy continuation
			item.contentLines = append(item.contentLines, strings.TrimLeft(ln, " \t"))
			i++
		}

		items = append(items, item)
	}

	isCheckList := len(items) > 0 && items[0].isCheck

	var itemNodes []*boxnote.Node
	for _, it := range items {
		subBlocks := parseBlocks(it.contentLines, 0, len(it.contentLines))
		if len(subBlocks) == 0 {
			subBlocks = []*boxnote.Node{{Type: "paragraph"}}
		}
		if isCheckList {
			itemNodes = append(itemNodes, &boxnote.Node{
				Type:    "check_list_item",
				Attrs:   jsonAttrs(map[string]any{"checked": it.checked}),
				Content: subBlocks,
			})
		} else {
			itemNodes = append(itemNodes, &boxnote.Node{
				Type:    "list_item",
				Content: subBlocks,
			})
		}
	}

	if isCheckList {
		return &boxnote.Node{Type: "check_list", Content: itemNodes}, i
	}
	if baseOrdered {
		return &boxnote.Node{
			Type:    "ordered_list",
			Attrs:   jsonAttrs(map[string]any{"order": baseOrder}),
			Content: itemNodes,
		}, i
	}
	return &boxnote.Node{Type: "bullet_list", Content: itemNodes}, i
}

// ============================================================
// テーブル
// ============================================================

func splitTableRow(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")
	var cells []string
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			buf.WriteByte(s[i+1])
			i++
			continue
		}
		if c == '|' {
			cells = append(cells, strings.TrimSpace(buf.String()))
			buf.Reset()
			continue
		}
		buf.WriteByte(c)
	}
	cells = append(cells, strings.TrimSpace(buf.String()))
	return cells
}

func parseTable(lines []string, start, end int) (*boxnote.Node, int) {
	header := splitTableRow(lines[start])
	colCount := len(header)
	i := start + 2
	rows := [][]string{header}
	for i < end {
		ln := lines[i]
		if !reTableRow.MatchString(ln) {
			break
		}
		cells := splitTableRow(ln)
		if len(cells) < colCount {
			for len(cells) < colCount {
				cells = append(cells, "")
			}
		} else if len(cells) > colCount {
			cells = cells[:colCount]
		}
		rows = append(rows, cells)
		i++
	}

	var tableRows []*boxnote.Node
	for _, r := range rows {
		var cellNodes []*boxnote.Node
		for _, ct := range r {
			var paragraphs []*boxnote.Node
			parts := []string{ct}
			if strings.Contains(ct, "<br><br>") {
				parts = strings.Split(ct, "<br><br>")
			}
			for _, p := range parts {
				inner := strings.ReplaceAll(p, "<br>", "\n")
				if strings.TrimSpace(inner) != "" {
					paragraphs = append(paragraphs, &boxnote.Node{
						Type:    "paragraph",
						Content: parseInline(inner),
					})
				} else {
					paragraphs = append(paragraphs, &boxnote.Node{Type: "paragraph"})
				}
			}
			cellNodes = append(cellNodes, &boxnote.Node{
				Type:    "table_cell",
				Attrs:   jsonAttrs(map[string]any{"colspan": 1, "rowspan": 1, "colwidth": nil}),
				Content: paragraphs,
			})
		}
		tableRows = append(tableRows, &boxnote.Node{Type: "table_row", Content: cellNodes})
	}
	return &boxnote.Node{Type: "table", Content: tableRows}, i
}

// ============================================================
// ユーティリティ
// ============================================================

func jsonAttrs(m map[string]any) json.RawMessage {
	b, _ := json.Marshal(m)
	return b
}

func numStr(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			break
		}
		n = n*10 + int(s[i]-'0')
	}
	return n
}

// lenExpand: タブ展開した上での文字幅を返す。
func lenExpand(s string) int {
	n := 0
	for _, c := range s {
		if c == '\t' {
			n += 4
		} else {
			n++
		}
	}
	return n
}

func leadingWS(s string) int {
	for i, c := range s {
		if c != ' ' && c != '\t' {
			return i
		}
	}
	return len(s)
}
