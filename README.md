# boxnote2md / md2boxnote

Box Note (`.boxnote`) ⇄ Markdown 双方向変換 CLI (Go 製、依存ゼロ・単一バイナリ)。

- `boxnote2md`: `.boxnote` → `.md`
- `md2boxnote`: `.md` → `.boxnote`

`.boxnote` は **新フォーマット (ProseMirror JSON, schema_version=1)** にのみ対応。
旧 Etherpad 形式 (`atext` + `pool`) は非対応。

## インストール

### バイナリ配布 (推奨)

[Releases](../../releases) から OS/Arch に合った tar / zip をダウンロード、展開して `PATH` 上に置くだけ。

| OS | Arch | バイナリ |
|---|---|---|
| Linux | amd64 | `dist/linux-amd64/{boxnote2md,md2boxnote}` |
| macOS (Intel) | amd64 | `dist/darwin-amd64/{boxnote2md,md2boxnote}` |
| macOS (Apple Silicon) | arm64 | `dist/darwin-arm64/{boxnote2md,md2boxnote}` |
| Windows | amd64 | `dist/windows-amd64/{boxnote2md.exe,md2boxnote.exe}` |

### ソースからビルド

要件: Go 1.22+ (動作確認は 1.26)。

```bash
git clone https://github.com/nanakanok/boxnote2md-cli.git
cd boxnote2md-cli
make build           # 現在の OS/Arch でビルド (dist/ 配下)
make build-all       # 4 ターゲットへクロスビルド
make install         # $GOBIN にインストール
```

### `go install`

```bash
go install github.com/nanakanok/boxnote2md-cli/cmd/boxnote2md@latest
go install github.com/nanakanok/boxnote2md-cli/cmd/md2boxnote@latest
```

## 使い方

### `boxnote2md` (.boxnote → .md)

```bash
# 単一ファイル
boxnote2md path/to/note.boxnote
# → ./out/note.md

# ディレクトリ (再帰)
boxnote2md path/to/box_drive/folder -o ./markdown
# → ./markdown/ 配下に元ディレクトリ構造を保ったまま .md を出力
```

### `md2boxnote` (.md → .boxnote)

```bash
# 単一ファイル
md2boxnote path/to/note.md
# → ./out/note.boxnote (Box Drive に置けば Box Note として開ける)

# ディレクトリ (再帰)
md2boxnote path/to/markdown -o ./boxnotes
```

引数はフラグ・位置引数のどちらの順でも OK (内部で自動並べ替え):

```bash
boxnote2md note.boxnote -o ./out -image-mode url -v
boxnote2md -o ./out -image-mode url -v note.boxnote
```

### `boxnote2md` の主なオプション

| オプション | 説明 |
|---|---|
| `-o`, `-out <dir>` | 出力先ディレクトリ (default: `./out`) |
| `-no-recursive` | ディレクトリ入力時の再帰探索を無効化 |
| `-flat` | 出力先直下にフラット配置 (同名衝突時は連番付与) |
| `-overwrite` | 既存 `.md` を上書き (default: スキップ) |
| `-image-mode {download,url}` | 画像の扱い (default: `download`) |
| `-image-dir <dir>` | 画像保存先 (default: `<out>/images`) |
| `-keep-styles` | `font_size` / `font_color` / `highlight` / `call_out_box` の背景色を HTML として残す |
| `-dry-run` | 書き込みせず処理予定だけ表示 |
| `-v` | 詳細ログ |

### `md2boxnote` の主なオプション

| オプション | 説明 |
|---|---|
| `-o`, `-out <dir>` | 出力先ディレクトリ (default: `./out`) |
| `-no-recursive` | ディレクトリ入力時の再帰探索を無効化 |
| `-flat` | 出力先直下にフラット配置 |
| `-overwrite` | 既存 `.boxnote` を上書き (default: スキップ) |
| `-dry-run` | 書き込みせず処理予定だけ表示 |
| `-v` | 詳細ログ |

### 画像

- `-image-mode download` (既定): `boxSharedLink` を取得し `<image-dir>/<boxFileId>__<fileName>` に保存。MD には相対パスで埋め込み。
- `-image-mode url`: 元の `boxSharedLink` をそのまま埋め込み。
- DL に失敗した場合は警告を出して URL 埋め込みにフォールバック。
- `boxSharedLink` が無いケースは `![image:fileName](#unavailable)` のプレースホルダ。

## 対応している ProseMirror ノード/マーク

### ブロック
- `paragraph`, `heading`, `horizontal_rule`
- `bullet_list`, `ordered_list` (`order` 反映), `check_list` (チェック状態反映)
- `blockquote`, `code_block` (言語フェンス)
- `call_out_box` → 既定では絵文字付き blockquote、`-keep-styles` で `<div style="background-color:...">`
- `table` / `table_row` / `table_cell` (改行→`<br>`、パイプエスケープ、colspan/rowspan は警告のみで 1×1)
- `image` / `box_preview`

### マーク
- `strong`, `em`, `underline`, `strikethrough`
- `link`
- `font_size`, `font_color`, `highlight` → 既定では破棄、`-keep-styles` で HTML span に退避
- `alignment` → MD 標準で表現できないため破棄
- `annotation_id`, `author_id` → 編集メタとして無視

## ラウンドトリップ (`.boxnote` ⇄ `.md`)

### 保持される情報
- 段落・heading・hr・blockquote・code_block (言語含む)
- bullet/ordered/check リスト (チェック状態・開始番号含む)
- table (1×1 セル)
- inline marks: strong / em / underline / strikethrough / link
- image (URL のみ)

### 失われる情報 (片方向のみ復元可)

| 元 (.boxnote 側) | md 経由後の状態 |
|---|---|
| `call_out_box` (絵文字+背景色) | 通常の `blockquote` に退化 |
| `box_preview` (Box ファイル埋め込み) | 通常の `link` に退化 |
| `font_size` / `font_color` / `highlight` | 既定では破棄 (`-keep-styles` で HTML 退避) |
| `alignment` | MD で表現できないため破棄 |
| `annotation_id` / `author_id` | 破棄 (Box 側で再付与される想定) |
| `image` の `boxFileId` / `boxSharedLink` メタ | URL 部分のみ保持 |

## 開発

```bash
make test       # ユニットテスト + ラウンドトリップテスト (35 ケース)
make lint       # go vet + gofmt
make build      # ローカルビルド
make build-all  # クロスビルド (4 ターゲット)
```

ディレクトリ構成:

```
boxnote2md-cli/
├── go.mod
├── Makefile
├── cmd/
│   ├── boxnote2md/main.go
│   └── md2boxnote/main.go
├── internal/
│   ├── boxnote/        # ProseMirror 型 + .boxnote I/O
│   ├── render/         # ProseMirror → Markdown
│   ├── mdparse/        # Markdown → ProseMirror (自作パーサ)
│   ├── runner/         # ファイル/ディレクトリ走査・実行
│   └── cliutil/        # flag 周りのユーティリティ
└── testdata/
    └── sample.boxnote
```

## 履歴

このリポはもともと Python で書かれ、配布性向上のため Go へ移植しました。
旧 Python 実装は git 履歴に残っています。

## ライセンス

MIT
