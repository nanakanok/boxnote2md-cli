# v0.1.0 — Initial Go release

Box Note (`.boxnote`) ⇄ Markdown 双方向変換 CLI の最初の Go 実装リリース。

## ハイライト

- **依存ゼロ・単一バイナリ**: 標準ライブラリのみで動作。インストール先で何も追加不要。
- **2 コマンド同梱**:
  - `boxnote2md`: `.boxnote` → `.md`
  - `md2boxnote`: `.md` → `.boxnote`
- **クロスプラットフォーム**: Linux / macOS (Intel + Apple Silicon) / Windows
- **ラウンドトリップ対応**: 主要ブロック (heading / list / code_block / table / image など) は同数復元

## バイナリ

下のアセットから OS/Arch に合ったアーカイブをダウンロード、展開して `PATH` に置くだけ。

| Asset | 中身 |
|---|---|
| `boxnote2md-cli-v0.1.0-linux-amd64.tar.gz` | linux/amd64 用バイナリ + README + LICENSE |
| `boxnote2md-cli-v0.1.0-darwin-amd64.tar.gz` | macOS Intel 用 |
| `boxnote2md-cli-v0.1.0-darwin-arm64.tar.gz` | macOS Apple Silicon 用 |
| `boxnote2md-cli-v0.1.0-windows-amd64.zip` | Windows 用 |
| `SHA256SUMS` | 各アーカイブの SHA256 ハッシュ |

## 使い方の例

```bash
# .boxnote → .md
boxnote2md path/to/note.boxnote -o ./out

# .md → .boxnote
md2boxnote path/to/note.md -o ./out

# ディレクトリを再帰変換
boxnote2md path/to/box_drive_folder -o ./markdown
```

詳細は `README.md` を参照。

## 既知の制限

- 旧 Etherpad 形式 (`atext` + `pool`) の `.boxnote` は非対応 (新 ProseMirror JSON 形式のみ)
- `call_out_box` / `box_preview` / `font_size` / `font_color` / `highlight` / `alignment` は MD 経由でラウンドトリップすると失われる (詳細: README)
