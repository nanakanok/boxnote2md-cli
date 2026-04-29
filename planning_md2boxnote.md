# Markdown → Box Note CLI — Planning

## 0. 確定事項

| 項目 | 決定 |
|---|---|
| Markdown パーサ | **自作** (標準ライブラリのみ・依存追加なし) |
| 配布形態 | **別コマンド `md2boxnote`** を `pyproject.toml` の `[project.scripts]` に追加 |
| 出力形式 | Box Note 新フォーマット (ProseMirror JSON, schema_version=1) |
| 対象 Markdown | 主に本ツール (`boxnote2md`) が出力する MD のラウンドトリップ。標準的な GFM 入力にも実用範囲で対応 |

## 1. 目的

ローカルの `.md` ファイル (またはディレクトリ配下) を Box Note 形式 `.boxnote` に変換する CLI を作る。`.boxnote` ファイルは Box Drive に置けば Box が取り込み可能 (公式の取り込みフォーマット仕様準拠を狙う)。

## 2. 出力する `.boxnote` の envelope

```json
{
  "version": 1,
  "schema_version": 1,
  "doc": {
    "type": "doc",
    "attrs": {
      "table_of_contents": { "enabled": false, "allowedLevels": [1, 2, 3] }
    },
    "content": [...]
  },
  "savepoint_metadata": {},
  "last_edit_timestamp": <epoch_ms (生成時刻)>
}
```

## 3. サポートする Markdown 構文

### 3.1 ブロック
| 記法 | 出力 ProseMirror ノード |
|---|---|
| `# ... ######` (ATX heading 1-6) | `heading {attrs:{level}}` |
| 段落 (連続する非空行) | `paragraph` |
| 空行 | 段落の区切り (空段落は基本生成しない) |
| `---` / `***` / `___` (3+) | `horizontal_rule` |
| `> line` (連続) | `blockquote` (中身を再帰パース) |
| ` ```lang\n...\n``` ` | `code_block {attrs:{language}}` |
| `- item` / `* item` / `+ item` | `bullet_list / list_item` |
| `1. item` (任意の N から開始) | `ordered_list {attrs:{order:N}} / list_item` |
| `- [ ] / - [x]` (item) | `check_list / check_list_item {attrs:{checked}}` |
| GFM テーブル (`|...|`) | `table / table_row / table_cell` (1 行目をヘッダ扱いの慣例どおり) |
| (image 単独段落) `![alt](url)` | `image` (attrs.src=url, alt, fileName=basename) |

### 3.2 インライン
| 記法 | 出力 |
|---|---|
| `**X**` | `text` + mark `strong` |
| `*X*` / `_X_` | `text` + mark `em` |
| `<u>X</u>` | `text` + mark `underline` (本ツールの emit と整合) |
| `~~X~~` | `text` + mark `strikethrough` |
| `[label](url)` | `text` (label をさらにインライン解析) + mark `link {href:url}` |
| `![alt](url)` | inline 内の `image` ノード (段落子) |
| `` `code` `` | `text` (バッククォートを残してそのままテキスト化。Box の inline code mark は確認できていないため、情報損失を避けて記号で残す) |
| 末尾 `  \n` (2スペース+改行) | `hard_break` |

### 3.3 取り扱わない / 退化させる
- Setext heading (`===` / `---` 下線記法) → 当面非対応 (paragraph + hr 扱いになる可能性あり)
- リンク参照定義 `[label]: url` → 非対応
- HTML 一般 (`<u>` を除く) → そのまま `text` として出力 (情報損失)
- 注釈・脚注・math → 非対応
- フロントマター (`---\n...\n---`) → **段落としてそのまま** (オプションで剥がすかは将来検討)
- `call_out_box`, `box_preview`, `font_size`, `font_color`, `highlight`, `alignment` → MD に表現が無いため復元不可。`call_out_box` は通常の blockquote に、`box_preview` は通常のリンクに退化する (これは既知の仕様)

## 4. CLI 仕様 (`md2boxnote`)

```
md2boxnote <input> [options]
```

| オプション | 既定 | 説明 |
|---|---|---|
| `<input>` | 必須 | `.md` ファイル または ディレクトリ |
| `-o / --out <dir>` | `./out` | 出力先ディレクトリ |
| `--no-recursive` | 再帰=有効 | ディレクトリ入力時の再帰探索を無効化 |
| `--flat` | false | 出力先直下にフラット配置 |
| `--overwrite` | false | 既存 `.boxnote` を上書き |
| `--dry-run` | false | 書き込みせず処理予定だけ表示 |
| `-v / --verbose` | false | 詳細ログ |
| `--version` / `-h` | - | 標準 |

### 動作
- ファイル入力 → `<out>/<basename>.boxnote`
- ディレクトリ入力 → 再帰収集 → 入力からの相対パスを保って `<out>` 配下に配置
- スキップ/上書き/サマリ/終了コードは `boxnote2md` と同等

## 5. アーキテクチャ

```
boxnote2md/
├── md/
│   ├── __init__.py
│   ├── parser.py          # 行ベースのブロックパーサ + インラインパーサ
│   └── builder.py         # AST → ProseMirror JSON envelope 化
├── md2boxnote_runner.py   # 入力解決・実行・サマリ
└── md2boxnote_cli.py      # argparse エントリポイント
```

- `parser.parse_markdown(text: str) -> dict` が ProseMirror の `doc` ノードを返す。
- `builder.wrap_envelope(doc: dict) -> dict` が Box Note のトップレベル JSON を組み立てる。

## 6. パーサ実装方針

### 6.1 ブロック層 (line-based)
1. `splitlines()` でリスト化
2. 単純な状態機械: 各行を見て該当ブロックの開始/継続/終了を判定
3. 多くのブロックは 1 行で確定 (heading, hr, blank)。複数行が絡むのは:
   - **段落**: 次の空行 or 別ブロック開始まで
   - **blockquote**: `>` で始まる行が続く限り (中身を再帰パース)
   - **code_block**: ` ``` ` で開始/終了
   - **list**: 同じインデント・同じ系統のマーカーが続く間 (項目内の継続行はインデントで判定)
   - **table**: ヘッダ + セパレータ + ボディ行
4. リストのネストは項目内コンテンツを再帰パースして実現

### 6.2 インライン層
1. **トークナイズ**: バックスラッシュエスケープ、`**`, `*`, `~~`, `<u>`, `</u>`, `` ` ``, `[`, `](`, `)`, `!`, `  \n`, リテラル文字を順に切り出す
2. **構文木化**: `**` `*` `~~` `<u>` を「open/close ペア」としてスタックで対応付け (greedy match)
3. **link/image**: `[text](url)` を最優先で確定し、内側 `text` を再帰インラインパース
4. **マーク適用**: 各 text セグメントに対し、その時点で開いているマークの集合を `marks: []` として付与
5. 不正な未閉ペア (`**foo`) は単なる literal 扱い

### 6.3 単純化する点 (既知の限界)
- `*foo *bar* baz*` のような曖昧な emphasis は最近接マッチで解決 (CommonMark の delimiter run 完全準拠はしない)
- 4 スペース indent コードブロック (旧式) は非対応 (フェンスのみ)
- `<>` 角括弧自動リンクは非対応

## 7. ラウンドトリップ保証 (目標)

`sample.boxnote → boxnote2md → md → md2boxnote → boxnote'` のとき、以下が一致することを目標とする (情報損失の有無):

| 要素 | ラウンドトリップ |
|---|---|
| 段落・heading・hr・blockquote・code_block・list (各種)・table・image | ◎ 構造一致 |
| inline marks (strong/em/u/strike/link) | ◎ 一致 |
| `call_out_box`, `box_preview` | △ 退化 (callout → blockquote、box_preview → link) |
| `font_size`, `font_color`, `highlight`, `alignment` | ✕ 失われる (`--keep-styles` 出力からの逆解釈は将来課題) |
| `annotation_id`, `author_id` | ✕ 失われる (Box 取り込み後に再付与される想定) |

## 8. 実装フェーズ

| Phase | 内容 |
|---|---|
| A | 本 planning |
| B | ブロックパーサ (heading/paragraph/hr/blockquote/code_block/lists) |
| C | テーブル + インラインパーサ |
| D | builder + runner + CLI |
| E | テスト + ラウンドトリップ + README 更新 |

## 9. リスクと留意点

- **Box の取り込み互換性**: 本ツールが生成した `.boxnote` を Box Drive に置いた際、Box 側の取り込みが厳格な場合は `version` フィールドの値や `attrs.table_of_contents` のフォーマット差異で弾かれる可能性。実機での回帰確認は別途。
- **エンコーディング**: 入力 MD は UTF-8 想定。BOM 付きの場合は剥がす。
- **巨大ファイル**: 行数で線形のため、極端な巨大ファイルでも問題ないはず。
- **不整合な Markdown**: 不正な構文は段落 (literal text) に落として黙殺。`-v` で警告。
