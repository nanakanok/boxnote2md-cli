# Box Note to Markdown CLI — Planning

## 0. ユーザー回答（確定事項）

| # | 質問 | 回答 |
|---|---|---|
| 1 | サンプル `.boxnote` の所在 | `app/src/boxnote-to-md/tests/sample.boxnote`（**新フォーマット**） |
| 2 | 言語 | **Python** で作成 |
| 3 | 画像の扱い | **ダウンロードしてローカル保存（既定）**。`--image-mode url` でURL埋め込み |
| 4 | 出力先デフォルト | `./out` |

## 1. 目的

Box Drive 上のローカル `.boxnote` ファイル（または `.boxnote` を含むディレクトリ）を入力に取り、Markdown (`.md`) を指定ディレクトリに書き出す Python 製 CLI を作る。

## 2. `.boxnote` の実フォーマット（新版・ProseMirror JSON）

### 2.1 トップレベル
```jsonc
{
  "version": 1460,
  "schema_version": 1,
  "doc": { "type": "doc", "attrs": {...}, "content": [...] },
  "savepoint_metadata": {...},
  "last_edit_timestamp": 1700000000000
}
```
変換に必要なのは **`doc` フィールドのみ**。他はメタデータで無視。

### 2.2 検出されたノード/マーク種別（`sample.boxnote` 実測）

#### ブロックノード
| type | attrs | 子 | Markdown 出力 |
|---|---|---|---|
| `doc` | `table_of_contents` | content[] | ルート |
| `heading` | `level` (1..6), `guid` | inline | `# / ## / ### ...` |
| `paragraph` | - | inline | 段落 + `\n\n` |
| `bullet_list` | - | `list_item[]` | `-` リスト |
| `ordered_list` | `order` (開始番号) | `list_item[]` | `1.` リスト（order から開始） |
| `list_item` | - | block | リスト項目 |
| `check_list` | - | `check_list_item[]` | タスクリスト |
| `check_list_item` | `checked` (bool) | block | `- [x]` / `- [ ]` |
| `table` | - | `table_row[]` | MDテーブル（先頭行ヘッダ扱い） |
| `table_row` | - | `table_cell[]` | テーブル行 |
| `table_cell` | `colspan`, `rowspan`, `colwidth` | block | セル（pipe エスケープ・改行→`<br>`） |
| `code_block` | `language` | text | ` ```lang ... ``` ` |
| `call_out_box` | `backgroundColor`, `emoji` | block | `> {emoji} {content}`（blockquote相当） |
| `blockquote` | - | block | `> ...` |
| `horizontal_rule` | - | - | `---` |
| `image` (atom) | `src`, `boxSharedLink`, `boxFileId`, `fileName`, `width`, `height`, `alt`, `title` | - | `![alt](path or url)` |
| `box_preview` (atom) | `boxSharedLink` | - | `[Box: filename](boxSharedLink)`（インライン or 段落として埋め込み） |

#### インライン
| type | 出力 |
|---|---|
| `text` (with `marks[]`) | テキスト + 各マーク適用 |

#### マーク
| type | attrs | 出力 |
|---|---|---|
| `strong` | - | `**...**` |
| `em` | - | `*...*` |
| `underline` | - | `<u>...</u>`（MD 標準なし） |
| `strikethrough` | - | `~~...~~` |
| `link` | `href` | `[...](href)` |
| `highlight` | `color` | `==...==`（拡張記法） or `<mark>...</mark>` |
| `font_size` | `size` (例: `0.8125em`) | **既定で無視**（情報損失。`--keep-styles` で `<span style="font-size:..."> `） |
| `font_color` | `color` | **既定で無視**（同上） |
| `alignment` | `alignment` (left/center/right) | **既定で無視**（MD 標準で表現不能。call_out_box / 段落属性的に出ることがある点に注意） |
| `annotation_id` | コメントID | 無視 |
| `author_id` | 著者ID | 無視（編集メタ） |

### 2.3 Box 固有ブロックの方針
- **image**: `boxSharedLink` を持つので、`download` 時はそれを叩いて取得、`url` 時はそのまま `![](url)` に埋め込む。`fileName` を `alt` の候補にする。`width/height` が指定されていれば `<img>` フォールバックも検討（既定は MD 記法）。
- **box_preview**: Box ファイルの埋め込みプレビュー。MD では再現不能なため、**リンクとして退避**: `[Box file: <basename(boxSharedLink)>](<boxSharedLink>)`。
- **call_out_box**: `> {emoji} ...` で `blockquote` に寄せる。`backgroundColor` は既定無視（`--keep-styles` で `<div style="background:...">`）。

### 2.4 既存拡張 (`src/boxnote-to-md/`) との関係
既存は `box.com` 上の HTML DOM を Markdown 化する Chrome 拡張。**入力形式が異なる**ため `content.js` のロジックは流用しないが、出力規約（heading/list/table/inline marks の Markdown 文字列化）の対応表は参考にする。

## 3. CLI 仕様

### 3.1 コマンド
```
boxnote2md <input> [options]
```

### 3.2 引数・オプション
| 引数/オプション | 型 | 既定 | 説明 |
|---|---|---|---|
| `<input>` | path | 必須 | `.boxnote` ファイル または ディレクトリ |
| `-o / --out` | path | `./out` | 出力先ディレクトリ |
| `-r / --recursive / --no-recursive` | flag | true | ディレクトリの再帰探索 |
| `--flat` | flag | false | 入力ディレクトリ構造を保持せず `<out>` 直下にフラット出力 |
| `--overwrite` | flag | false | 既存 `.md` を上書き（既定: skip） |
| `--image-mode` | choice | `download` | `download` / `url` |
| `--image-dir` | path | `<out>/images` | `download` 時の保存先 |
| `--keep-styles` | flag | false | font_size/font_color/highlight/alignment を HTML span/div で残す |
| `--dry-run` | flag | false | 書き込みせず処理計画を表示 |
| `-v / --verbose` | flag | false | 詳細ログ |
| `--version` | flag | - | バージョン |

### 3.3 動作
- ファイル入力 → `<out>/<basename>.md` 生成（拡張子置換）。
- ディレクトリ入力 → 再帰収集 → 入力からの相対パスを `<out>` 配下に再現（`--flat` 時は直下、衝突時は連番 `name-1.md`）。
- 名前にスペースを含むファイルも正しく扱う。
- 終了サマリ: `success / skipped / failed`。失敗があれば exit code 1。

### 3.4 画像の取り扱い
- `image` ノードの `boxSharedLink` を最優先で利用。
- `--image-mode download`（既定）:
  - `boxSharedLink` を `httpx` で GET → `<image-dir>/<boxFileId>__<fileName>` に保存。
  - MD 上は `<image-dir>` への**出力 .md からの相対パス**で `![alt](path)`。
  - 失敗時は警告 + URL フォールバック。
- `--image-mode url`: `boxSharedLink` をそのまま `![alt](url)` に。
- `boxSharedLink` が空かつ `boxFileId` のみのケース: API なしでは取得不可 → プレースホルダ `![image:fileName](#unavailable)`。

## 4. 技術選定（Python）

| 項目 | 採用 | 理由 |
|---|---|---|
| Python | 3.11+ | 型ヒント・パターンマッチ・`tomllib` |
| CLI パーサ | **`click`** | option/argument/choice の表現力 |
| パッケージ管理 | `pyproject.toml` (PEP 621) | 余計な依存を増やさない |
| HTTP（画像DL） | `httpx`（同期使用） | タイムアウト・リトライが書きやすい |
| テスト | `pytest` | デファクト |
| Lint/Format | `ruff` | 一発で済む |

外部依存は最小（`click`, `httpx`, `pytest`, `ruff`）。

## 5. ディレクトリ構成

```
src/boxnote-to-md-cli/
├── pyproject.toml
├── README.md
├── planning.md                       ← 本ファイル
├── boxnote2md/
│   ├── __init__.py
│   ├── __main__.py                   # `python -m boxnote2md`
│   ├── cli.py                        # click エントリポイント
│   ├── runner.py                     # 入力解決・実行・サマリ
│   ├── reader.py                     # .boxnote 読込・JSON parse・doc抽出
│   ├── render/
│   │   ├── __init__.py
│   │   ├── document.py               # doc → Markdown ディスパッチャ
│   │   ├── blocks.py                 # heading/paragraph/list/table/blockquote/code/hr/callout
│   │   ├── inline.py                 # text + marks (strong/em/u/s/link/highlight/...)
│   │   ├── tables.py                 # table_row/table_cell の整形（pipe / <br>）
│   │   └── images.py                 # image/box_preview のDL・パス解決
│   └── writer.py                     # 出力先解決・上書き制御
└── tests/
    ├── fixtures/                     # sample.boxnote をリンク or コピー
    ├── test_render_inline.py
    ├── test_render_blocks.py
    ├── test_render_lists.py
    ├── test_render_tables.py
    ├── test_render_images.py
    └── test_e2e.py                   # sample.boxnote 全体のスナップショット
```

## 6. 変換アルゴリズム（中核）

ProseMirror の **ビジターパターン** で実装。

```python
def render_node(node: dict, ctx: Context) -> str:
    t = node["type"]
    return DISPATCH[t](node, ctx)   # 未知 type は warn して content を素通し
```

- **block ノード**: 自身の前後に空行を挿入（リスト・表内では抑止）。
- **inline (text + marks)**: 連続するテキストノードに対し、マーク差分を最小限の `**`/`*`/`<u>`/`~~` で囲む。Wrap 順序は `link → strong → em → underline → strikethrough → highlight` を採用（読みやすさと閉じタグ整合のため）。
- **list_item / check_list_item**: 子ブロックの先頭段落をリストマーカーに連結し、2 段目以降は **4 スペースインデント**で続ける（GFM）。
- **ネスト**: `bullet_list/ordered_list/check_list` が `list_item` の中に再帰的に現れる構造をそのまま MD のインデント深さに反映。
- **table**: 1 行目をヘッダ扱い、セル内改行は `<br>`、セル内パイプは `\|` にエスケープ。`colspan/rowspan` は MD 非対応のため**警告して 1×1 として描画**。
- **image**: `images.py` で DL・パス解決し、Markdown 文字列を返す。

## 7. 実装フェーズ

| Phase | 内容 | 完了条件 |
|---|---|---|
| 0 | プロジェクト雛形（`pyproject.toml`, click 雛形） | `python -m boxnote2md --help` |
| 1 | `reader.py` + `render/document.py` 骨格 + テキスト/段落/heading/インラインマーク基本（strong/em/u/s/link） | `sample.boxnote` の冒頭〜heading 部分が期待MDに |
| 2 | リスト（bullet / ordered + order / check_list + checked） | `sample.boxnote` のリスト部分一致 |
| 3 | ブロック要素 (blockquote, code_block, horizontal_rule, call_out_box) | 該当ブロックが期待MDに |
| 4 | テーブル（pipe 整形・改行→`<br>`・パイプエスケープ・colspan警告） | `sample.boxnote` のテーブル部分一致 |
| 5 | 画像 (`image`) と `box_preview`（DL / URL モード / フォールバック） | 画像が `<out>/images/` に保存され、相対参照される |
| 6 | ディレクトリ再帰・出力構造保持・skip/overwrite・サマリ・終了コード | 複数ファイル一括処理 |
| 7 | README / 使用例 / `pyproject` の console_scripts | `pip install -e .` → `boxnote2md <dir>` 動作 |

## 8. リスク・留意点

- **`alignment` / `font_size` / `font_color` / `highlight`**: MD に 1:1 対応がない。**既定では落とす**。`--keep-styles` で HTML span/div 出力に拡張。
- **`call_out_box`** / **`box_preview`**: Box 独自。本実装では blockquote / リンクへの **意味的退避** とし、見た目は再現しない。
- **`colspan/rowspan`**: MD 非対応 → 警告ログのみ。
- **画像 DL**: `boxSharedLink` が公開リンクでない場合 401。`--image-mode url` を案内。
- **未知の type**: 将来的に増える可能性 → `DISPATCH` に無いタイプは warn を出して `content` を素通し（壊れたMDではなく欠損しただけのMDを出す）。
- **既存拡張との分離**: `src/boxnote-to-md-cli/` を独立ディレクトリ。

## 9. 次のアクション

- 本 planning の合意が取れたら **Phase 0**（雛形作成）に着手。
- Phase 1 と並行して `sample.boxnote` の期待 MD を**スナップショットの黄金値**として確定。
