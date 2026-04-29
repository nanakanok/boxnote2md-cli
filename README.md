# boxnote2md / md2boxnote

Box Note (`.boxnote`) ⇄ Markdown 双方向変換 CLI。

- `boxnote2md`: `.boxnote` → `.md`
- `md2boxnote`: `.md` → `.boxnote`

`.boxnote` は **新フォーマット (ProseMirror JSON, schema_version=1)** にのみ対応。
旧 Etherpad 形式 (`atext` + `pool`) は非対応。

## 必要環境

### システム要件

- Python **3.11 以上** (動作確認は 3.14)
- Linux / macOS (Windows は未検証)

### 必要なシステムパッケージ (Debian / Ubuntu)

| パッケージ | 用途 | 必須/任意 |
|---|---|---|
| `python3` | ランタイム | 必須 |
| `python3-venv` | 仮想環境作成 (`python3 -m venv`) | 推奨 |
| `python3-pip` | パッケージ導入 (`venv` 内で `pip` を使う場合は不要) | 任意 |

Debian/Ubuntu でセットアップする場合の例:

```bash
sudo apt-get update
sudo apt-get install -y python3 python3-venv
```

> **注意:** Debian/Ubuntu の最近のディストリは PEP 668 で「外部管理環境」になっており、システム Python に直接 `pip install` するとエラーになります。**必ず venv を使ってください**。

### Python パッケージ

| パッケージ | 用途 | バージョン |
|---|---|---|
| (なし) | **ランタイム依存はゼロ** — 標準ライブラリ (`argparse`, `urllib`, `json`, `pathlib`) のみで動作 | - |
| `pytest` | テスト実行 (任意) | `>=8` |
| `ruff` | Lint/Format (任意) | `>=0.5` |

## インストール

### 推奨: venv を使う

```bash
cd src/boxnote-to-md-cli
python3 -m venv .venv
.venv/bin/pip install -e .
```

これで `.venv/bin/boxnote2md` コマンドが利用可能になります。

### 開発時 (pytest, ruff も入れる)

```bash
.venv/bin/pip install -e ".[dev]"
```

### venv を使わずに直接実行 (依存追加なしで動かす)

ランタイム依存がゼロなので、リポジトリをそのまま `python3 -m` で起動できます:

```bash
PYTHONPATH=src/boxnote-to-md-cli python3 -m boxnote2md <input> -o ./out
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

### `boxnote2md` の主なオプション

| オプション | 説明 |
|---|---|
| `-o, --out <dir>` | 出力先ディレクトリ (default: `./out`) |
| `--no-recursive` | ディレクトリ入力時の再帰探索を無効化 |
| `--flat` | 出力先直下にフラット配置 (同名衝突時は連番付与) |
| `--overwrite` | 既存 `.md` を上書き (default: スキップ) |
| `--image-mode {download,url}` | 画像の扱い (default: `download`) |
| `--image-dir <dir>` | 画像保存先 (default: `<out>/images`) |
| `--keep-styles` | `font_size` / `font_color` / `highlight` / `call_out_box` の背景色を HTML として残す |
| `--dry-run` | 書き込みせず処理予定だけ表示 |
| `-v, --verbose` | 詳細ログ |

### `md2boxnote` の主なオプション

| オプション | 説明 |
|---|---|
| `-o, --out <dir>` | 出力先ディレクトリ (default: `./out`) |
| `--no-recursive` | ディレクトリ入力時の再帰探索を無効化 |
| `--flat` | 出力先直下にフラット配置 |
| `--overwrite` | 既存 `.boxnote` を上書き (default: スキップ) |
| `--dry-run` | 書き込みせず処理予定だけ表示 |
| `-v, --verbose` | 詳細ログ |

### 画像

- `--image-mode download` (既定): `boxSharedLink` を取得し `<image-dir>/<boxFileId>__<fileName>` に保存。MD には相対パスで埋め込み。
- `--image-mode url`: 元の `boxSharedLink` をそのまま埋め込み。
- DL に失敗した場合は警告を出して URL 埋め込みにフォールバック。
- `boxSharedLink` が無いケースは `![image:fileName](#unavailable)` のプレースホルダ。

## 対応している ProseMirror ノード/マーク

### ブロック
- `paragraph`, `heading`, `horizontal_rule`
- `bullet_list`, `ordered_list` (`order` 反映), `check_list` (チェック状態反映)
- `blockquote`, `code_block` (言語フェンス)
- `call_out_box` → 既定では絵文字付き blockquote、`--keep-styles` で `<div style="background-color:...">`
- `table` / `table_row` / `table_cell` (改行→`<br>`、パイプエスケープ、colspan/rowspan は警告のみで 1×1)
- `image` / `box_preview`

### マーク
- `strong`, `em`, `underline`, `strikethrough`
- `link`
- `font_size`, `font_color`, `highlight` → 既定では破棄、`--keep-styles` で HTML span に退避
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
| `font_size` / `font_color` / `highlight` | 既定では破棄 (`--keep-styles` で HTML 退避) |
| `alignment` | MD で表現できないため破棄 |
| `annotation_id` / `author_id` | 破棄 (Box 側で再付与される想定) |
| `image` の `boxFileId` / `boxSharedLink` メタ | URL 部分のみ保持 |

### サンプル経由のラウンドトリップ確認

```bash
# .boxnote → .md
boxnote2md src/boxnote-to-md/tests/sample.boxnote -o /tmp/md --image-mode url

# .md → .boxnote
md2boxnote /tmp/md/sample.md -o /tmp/back
```

主要ブロック (heading, list, code_block, table, image など) は同数復元される。

## 既存拡張との関係

`src/boxnote-to-md/` は Box Note ページを開いている Chrome 上で DOM を Markdown 化する Chrome 拡張。本 CLI はローカルの `.boxnote` ファイル (ProseMirror JSON) を直接変換するため、入出力経路は完全に独立。

## 開発

```bash
.venv/bin/pytest tests/                       # テスト (現在 81 ケース)
.venv/bin/ruff check boxnote2md tests         # Lint
.venv/bin/ruff format boxnote2md tests        # Format
```

## トラブルシュート

### `python3 -m venv` が `Failing command: ...` で失敗する

→ `python3-venv` パッケージが入っていません:

```bash
sudo apt-get install -y python3-venv
```

その後、既に作りかけの `.venv` を削除して再実行:

```bash
rm -rf .venv
python3 -m venv .venv
```

### `error: externally-managed-environment` で `pip install` が失敗する

→ Debian/Ubuntu の PEP 668 保護です。**venv を作って、その中の pip を使ってください** (上の「インストール」参照)。`--break-system-packages` は推奨しません (システム Python を汚染するため)。

### 画像 DL が 401 / 403 で失敗する

→ Box の共有リンク (`boxSharedLink`) が認証必要なリンクの場合、未認証ではダウンロードできません。`--image-mode url` でリンク埋め込みに切り替えるか、ブラウザ等で個別に取得してください。

## ライセンス

MIT
