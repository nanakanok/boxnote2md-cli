// Package boxnote は Box Note (.boxnote) ファイル形式 (新スキーマ ProseMirror JSON) の
// 型定義と読み書きを提供する。
package boxnote

import "encoding/json"

// Envelope は .boxnote ファイルのトップレベル JSON。
type Envelope struct {
	Version           int             `json:"version"`
	SchemaVersion     int             `json:"schema_version"`
	Doc               *Node           `json:"doc"`
	SavepointMetadata json.RawMessage `json:"savepoint_metadata,omitempty"`
	LastEditTimestamp int64           `json:"last_edit_timestamp,omitempty"`
}

// Node は ProseMirror のノード/インラインを汎用に表現する。
//
// 全ての type を構造体化するのは既知/未知の type が混在するためコスト高なので、
// attrs と marks は json.RawMessage で抱え、必要に応じて parser/renderer で
// 型キャストする方針。
type Node struct {
	Type    string          `json:"type"`
	Attrs   json.RawMessage `json:"attrs,omitempty"`
	Content []*Node         `json:"content,omitempty"`
	Marks   []*Mark         `json:"marks,omitempty"`
	Text    string          `json:"text,omitempty"`
}

// Mark は ProseMirror のインラインマーク。
type Mark struct {
	Type  string          `json:"type"`
	Attrs json.RawMessage `json:"attrs,omitempty"`
}

// AttrInt は attrs から整数を取り出す簡易ユーティリティ。
// 取得できない場合は def を返す。
func (n *Node) AttrInt(key string, def int) int {
	if len(n.Attrs) == 0 {
		return def
	}
	var m map[string]any
	if err := json.Unmarshal(n.Attrs, &m); err != nil {
		return def
	}
	v, ok := m[key]
	if !ok {
		return def
	}
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	default:
		return def
	}
}

// AttrString は attrs から文字列を取り出す。取得できない場合は空文字。
func (n *Node) AttrString(key string) string {
	if len(n.Attrs) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(n.Attrs, &m); err != nil {
		return ""
	}
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}

// AttrBool は attrs から真偽値を取り出す。取得できない場合は def を返す。
func (n *Node) AttrBool(key string, def bool) bool {
	if len(n.Attrs) == 0 {
		return def
	}
	var m map[string]any
	if err := json.Unmarshal(n.Attrs, &m); err != nil {
		return def
	}
	if b, ok := m[key].(bool); ok {
		return b
	}
	return def
}

// MarkAttrString は mark の attrs から文字列を取り出す。
func (m *Mark) AttrString(key string) string {
	if len(m.Attrs) == 0 {
		return ""
	}
	var attrs map[string]any
	if err := json.Unmarshal(m.Attrs, &attrs); err != nil {
		return ""
	}
	if s, ok := attrs[key].(string); ok {
		return s
	}
	return ""
}
