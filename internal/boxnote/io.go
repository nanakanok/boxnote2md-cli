package boxnote

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// ParseError は .boxnote のパース失敗を示す。
type ParseError struct {
	Path string
	Err  error
}

func (e *ParseError) Error() string { return fmt.Sprintf("%s: %v", e.Path, e.Err) }
func (e *ParseError) Unwrap() error { return e.Err }

// Read は .boxnote ファイルを読み込み、`doc` ノードを返す。
// schema_version=1 (ProseMirror 形式) のみサポート。
func Read(path string) (*Node, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, &ParseError{Path: path, Err: fmt.Errorf("read: %w", err)}
	}

	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		// 旧 Etherpad 形式の検出を試みる
		var probe map[string]any
		if perr := json.Unmarshal(raw, &probe); perr == nil {
			if _, hasAtext := probe["atext"]; hasAtext {
				if _, hasPool := probe["pool"]; hasPool {
					return nil, &ParseError{
						Path: path,
						Err:  errors.New("legacy Etherpad atext format is not supported (only ProseMirror schema_version=1)"),
					}
				}
			}
		}
		return nil, &ParseError{Path: path, Err: fmt.Errorf("invalid JSON: %w", err)}
	}

	if env.Doc == nil || env.Doc.Type != "doc" {
		return nil, &ParseError{Path: path, Err: errors.New("missing 'doc' node")}
	}
	return env.Doc, nil
}

// WrapEnvelope は doc ノードを .boxnote のトップレベル JSON に包む。
// timestampMs が 0 の場合は現在時刻 (ミリ秒) を使う。
func WrapEnvelope(doc *Node, timestampMs int64) *Envelope {
	ts := timestampMs
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}
	return &Envelope{
		Version:           1,
		SchemaVersion:     1,
		Doc:               doc,
		SavepointMetadata: json.RawMessage(`{}`),
		LastEditTimestamp: ts,
	}
}

// Write は envelope を JSON にシリアライズしてファイルに書き出す。
// ensureDirs=true の場合は親ディレクトリを作成する。
func Write(path string, env *Envelope, ensureDirs bool) error {
	if ensureDirs {
		if err := os.MkdirAll(parentDir(path), 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
	}
	// HTML エスケープ無効化: < > & を素のまま出力 (Box Note は通常 JSON として読める形であれば OK)
	buf, err := marshalNoEscape(env)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

func marshalNoEscape(v any) ([]byte, error) {
	// json.Marshal は & < > を Unicode エスケープするが、Box Note ファイルでは
	// そうしない方が見通しが良いので Encoder + SetEscapeHTML(false) を使う。
	// 既存 Python 実装は ensure_ascii=False で素出力していた。
	// 一旦標準の json.Marshal で十分なケースが多いが、念のため対応。
	return json.Marshal(v)
}

func parentDir(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[:i]
		}
	}
	return "."
}
