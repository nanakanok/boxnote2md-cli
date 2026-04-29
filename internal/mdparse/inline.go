package mdparse

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/nanakanok/boxnote2md-cli/internal/boxnote"
)

// インライントークン種別
const (
	tText      = "text"
	tStrong    = "strong_marker"
	tEm        = "em_marker"
	tStrike    = "strike_marker"
	tUOpen     = "u_open"
	tUClose    = "u_close"
	tHardBreak = "hard_break"
	tLink      = "link"  // value: linkToken{tokens, url}
	tImage     = "image" // value: imageToken{alt, url}
)

type inlineToken struct {
	kind  string
	text  string
	url   string
	inner []inlineToken
}

func parseInline(text string) []*boxnote.Node {
	tokens := tokenizeInline(text)
	return flattenTokens(tokens)
}

func tokenizeInline(text string) []inlineToken {
	var tokens []inlineToken
	n := len(text)
	i := 0
	for i < n {
		c := text[i]

		// backslash escape
		if c == '\\' && i+1 < n {
			tokens = append(tokens, inlineToken{kind: tText, text: string(text[i+1])})
			i += 2
			continue
		}
		// hard break "  \n"
		if c == ' ' && i+2 < n && text[i+1] == ' ' && text[i+2] == '\n' {
			tokens = append(tokens, inlineToken{kind: tHardBreak})
			i += 3
			continue
		}
		// image: ![alt](url)
		if c == '!' && i+1 < n && text[i+1] == '[' {
			if alt, u, end, ok := tryConsumeImage(text, i); ok {
				tokens = append(tokens, inlineToken{kind: tImage, text: alt, url: u})
				i = end
				continue
			}
		}
		// link: [...](url)
		if c == '[' {
			if inner, u, end, ok := tryConsumeLink(text, i); ok {
				tokens = append(tokens, inlineToken{kind: tLink, inner: inner, url: u})
				i = end
				continue
			}
		}
		// <u>, </u>
		if i+3 <= n && strings.EqualFold(text[i:i+3], "<u>") {
			tokens = append(tokens, inlineToken{kind: tUOpen})
			i += 3
			continue
		}
		if i+4 <= n && strings.EqualFold(text[i:i+4], "</u>") {
			tokens = append(tokens, inlineToken{kind: tUClose})
			i += 4
			continue
		}
		// strong / em
		if i+2 <= n && text[i:i+2] == "**" {
			tokens = append(tokens, inlineToken{kind: tStrong})
			i += 2
			continue
		}
		if c == '*' {
			tokens = append(tokens, inlineToken{kind: tEm})
			i++
			continue
		}
		// _emphasis_ (単語境界の場合のみ)
		if c == '_' {
			var prev byte = ' '
			if i > 0 {
				prev = text[i-1]
			}
			var next byte = ' '
			if i+1 < n {
				next = text[i+1]
			}
			if !(isAlnum(prev) && isAlnum(next)) {
				tokens = append(tokens, inlineToken{kind: tEm})
				i++
				continue
			}
		}
		// ~~strike~~
		if i+2 <= n && text[i:i+2] == "~~" {
			tokens = append(tokens, inlineToken{kind: tStrike})
			i += 2
			continue
		}
		// plain text
		tokens = append(tokens, inlineToken{kind: tText, text: string(c)})
		i++
	}
	return coalesceText(tokens)
}

func coalesceText(tokens []inlineToken) []inlineToken {
	var out []inlineToken
	for _, t := range tokens {
		if t.kind == tText && len(out) > 0 && out[len(out)-1].kind == tText {
			out[len(out)-1].text += t.text
		} else {
			out = append(out, t)
		}
	}
	return out
}

func tryConsumeImage(text string, start int) (alt, urlStr string, end int, ok bool) {
	bracketEnd := findMatchingBracket(text, start+1)
	if bracketEnd < 0 {
		return "", "", 0, false
	}
	if bracketEnd+1 >= len(text) || text[bracketEnd+1] != '(' {
		return "", "", 0, false
	}
	alt = text[start+2 : bracketEnd]
	urlStr, parenEnd, urlOK := consumeURL(text, bracketEnd+2)
	if !urlOK {
		return "", "", 0, false
	}
	return alt, urlStr, parenEnd + 1, true
}

func tryConsumeLink(text string, start int) (inner []inlineToken, urlStr string, end int, ok bool) {
	bracketEnd := findMatchingBracket(text, start)
	if bracketEnd < 0 {
		return nil, "", 0, false
	}
	if bracketEnd+1 >= len(text) || text[bracketEnd+1] != '(' {
		return nil, "", 0, false
	}
	label := text[start+1 : bracketEnd]
	urlStr, parenEnd, urlOK := consumeURL(text, bracketEnd+2)
	if !urlOK {
		return nil, "", 0, false
	}
	return tokenizeInline(label), urlStr, parenEnd + 1, true
}

func findMatchingBracket(text string, openPos int) int {
	depth := 1
	i := openPos + 1
	for i < len(text) {
		c := text[i]
		if c == '\\' && i+1 < len(text) {
			i += 2
			continue
		}
		if c == '[' {
			depth++
		} else if c == ']' {
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return -1
}

func consumeURL(text string, start int) (string, int, bool) {
	depth := 1
	i := start
	var buf strings.Builder
	for i < len(text) && depth > 0 {
		c := text[i]
		if c == '\\' && i+1 < len(text) {
			buf.WriteByte(text[i+1])
			i += 2
			continue
		}
		if c == '(' {
			depth++
			buf.WriteByte(c)
		} else if c == ')' {
			depth--
			if depth == 0 {
				break
			}
			buf.WriteByte(c)
		} else {
			buf.WriteByte(c)
		}
		i++
	}
	if depth != 0 {
		return "", i, false
	}
	return strings.TrimSpace(buf.String()), i, true
}

// classifyMarkers: 各 marker トークン (strong/em/strike + u_open/u_close) を
// open/close ペアにする。ペアになれなかった marker は role 無し → literal 化。
type markerRole struct {
	markType string
	opening  bool
}

func classifyMarkers(tokens []inlineToken) map[int]markerRole {
	roles := map[int]markerRole{}
	stacks := map[string][]int{}
	for idx, tok := range tokens {
		switch tok.kind {
		case tStrong:
			if len(stacks["strong"]) > 0 {
				openIdx := stacks["strong"][len(stacks["strong"])-1]
				stacks["strong"] = stacks["strong"][:len(stacks["strong"])-1]
				roles[openIdx] = markerRole{"strong", true}
				roles[idx] = markerRole{"strong", false}
			} else {
				stacks["strong"] = append(stacks["strong"], idx)
			}
		case tEm:
			if len(stacks["em"]) > 0 {
				openIdx := stacks["em"][len(stacks["em"])-1]
				stacks["em"] = stacks["em"][:len(stacks["em"])-1]
				roles[openIdx] = markerRole{"em", true}
				roles[idx] = markerRole{"em", false}
			} else {
				stacks["em"] = append(stacks["em"], idx)
			}
		case tStrike:
			if len(stacks["strikethrough"]) > 0 {
				openIdx := stacks["strikethrough"][len(stacks["strikethrough"])-1]
				stacks["strikethrough"] = stacks["strikethrough"][:len(stacks["strikethrough"])-1]
				roles[openIdx] = markerRole{"strikethrough", true}
				roles[idx] = markerRole{"strikethrough", false}
			} else {
				stacks["strikethrough"] = append(stacks["strikethrough"], idx)
			}
		case tUOpen:
			stacks["underline"] = append(stacks["underline"], idx)
		case tUClose:
			if len(stacks["underline"]) > 0 {
				openIdx := stacks["underline"][len(stacks["underline"])-1]
				stacks["underline"] = stacks["underline"][:len(stacks["underline"])-1]
				roles[openIdx] = markerRole{"underline", true}
				roles[idx] = markerRole{"underline", false}
			}
		}
	}
	return roles
}

func markerLiteral(kind string) string {
	switch kind {
	case tStrong:
		return "**"
	case tEm:
		return "*"
	case tStrike:
		return "~~"
	case tUOpen:
		return "<u>"
	case tUClose:
		return "</u>"
	}
	return ""
}

func textNode(text string, marks []*boxnote.Mark) *boxnote.Node {
	n := &boxnote.Node{Type: "text", Text: text}
	if len(marks) > 0 {
		n.Marks = marks
	}
	return n
}

func currentMarks(stack []string) []*boxnote.Mark {
	marks := make([]*boxnote.Mark, 0, len(stack))
	for _, m := range stack {
		marks = append(marks, &boxnote.Mark{Type: m})
	}
	return marks
}

func flattenTokens(tokens []inlineToken) []*boxnote.Node {
	roles := classifyMarkers(tokens)
	var nodes []*boxnote.Node
	var stack []string

	for idx, tok := range tokens {
		switch tok.kind {
		case tText:
			if tok.text != "" {
				nodes = append(nodes, textNode(tok.text, currentMarks(stack)))
			}
		case tHardBreak:
			nodes = append(nodes, &boxnote.Node{Type: "hard_break"})
		case tImage:
			fileName := basenameFromURL(tok.url)
			if fileName == "" {
				fileName = tok.text
			}
			if fileName == "" {
				fileName = "image"
			}
			imgAttrs, _ := json.Marshal(map[string]any{
				"src":              tok.url,
				"alt":              tok.text,
				"title":            "",
				"boxSharedLink":    "",
				"boxFileId":        "",
				"fileName":         fileName,
				"placeholderState": "",
				"width":            nil,
				"height":           nil,
			})
			nodes = append(nodes, &boxnote.Node{
				Type:  "image",
				Attrs: imgAttrs,
			})
		case tLink:
			innerNodes := flattenTokens(tok.inner)
			outer := stack
			for _, ni := range innerNodes {
				if ni.Type == "text" {
					marks := append([]*boxnote.Mark{}, ni.Marks...)
					for _, m := range outer {
						if !hasMark(marks, m) {
							marks = append(marks, &boxnote.Mark{Type: m})
						}
					}
					linkAttrs, _ := json.Marshal(map[string]any{"href": tok.url})
					marks = append(marks, &boxnote.Mark{Type: "link", Attrs: linkAttrs})
					ni.Marks = marks
				}
				nodes = append(nodes, ni)
			}
		case tStrong, tEm, tStrike, tUOpen, tUClose:
			role, ok := roles[idx]
			if !ok {
				nodes = append(nodes, textNode(markerLiteral(tok.kind), currentMarks(stack)))
				continue
			}
			if role.opening {
				stack = append(stack, role.markType)
			} else {
				stack = removeMark(stack, role.markType)
			}
		}
	}
	return mergeAdjacentText(nodes)
}

func hasMark(marks []*boxnote.Mark, t string) bool {
	for _, m := range marks {
		if m.Type == t {
			return true
		}
	}
	return false
}

func removeMark(stack []string, t string) []string {
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] == t {
			return append(stack[:i], stack[i+1:]...)
		}
	}
	return stack
}

func mergeAdjacentText(nodes []*boxnote.Node) []*boxnote.Node {
	var out []*boxnote.Node
	for _, n := range nodes {
		if len(out) > 0 && n.Type == "text" && out[len(out)-1].Type == "text" && marksEqual(out[len(out)-1].Marks, n.Marks) {
			out[len(out)-1].Text += n.Text
		} else {
			out = append(out, n)
		}
	}
	return out
}

func marksEqual(a, b []*boxnote.Mark) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Type != b[i].Type {
			return false
		}
		if string(a[i].Attrs) != string(b[i].Attrs) {
			return false
		}
	}
	return true
}

func isAlnum(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func basenameFromURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parts := strings.Split(u.Path, "/")
	return parts[len(parts)-1]
}
