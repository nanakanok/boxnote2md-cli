// Command boxnote2md converts Box Note (.boxnote) files to Markdown.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nanakanok/boxnote2md-cli/internal/cliutil"
	"github.com/nanakanok/boxnote2md-cli/internal/runner"
)

const version = "0.1.0"

func usage() {
	fmt.Fprintf(os.Stderr, `boxnote2md %s — Box Note (.boxnote) を Markdown に変換する CLI

Usage:
  boxnote2md <input> [options]

Arguments:
  <input>            .boxnote ファイル または ディレクトリ

Options:
  -o, -out <dir>     出力先ディレクトリ (default: ./out)
  -no-recursive      ディレクトリ入力時の再帰探索を無効化
  -flat              出力先直下にフラット配置
  -overwrite         既存 .md を上書き (default: スキップ)
  -image-mode <m>    画像の扱い: download (default) / url
  -image-dir <dir>   画像保存先 (default: <out>/images)
  -keep-styles       font_size/font_color/highlight/alignment を HTML で残す
  -dry-run           書き込みせず処理予定だけ表示
  -v                 詳細ログ
  -version           バージョン表示
  -h                 ヘルプ

`, version)
}

func main() {
	fs := flag.NewFlagSet("boxnote2md", flag.ContinueOnError)
	fs.Usage = usage
	fs.SetOutput(os.Stderr)

	out := fs.String("out", "./out", "出力先ディレクトリ")
	fs.StringVar(out, "o", "./out", "出力先ディレクトリ (短縮形)")
	noRecursive := fs.Bool("no-recursive", false, "再帰探索を無効化")
	flat := fs.Bool("flat", false, "フラット配置")
	overwrite := fs.Bool("overwrite", false, "既存 .md を上書き")
	imageMode := fs.String("image-mode", "download", "画像の扱い (download|url)")
	imageDir := fs.String("image-dir", "", "画像保存先 (default: <out>/images)")
	keepStyles := fs.Bool("keep-styles", false, "プレゼンテーション系マークを HTML で残す")
	dryRun := fs.Bool("dry-run", false, "書き込みせず処理予定だけ表示")
	verbose := fs.Bool("v", false, "詳細ログ")
	showVersion := fs.Bool("version", false, "バージョン表示")

	if err := fs.Parse(cliutil.ReorderArgs(fs, os.Args[1:])); err != nil {
		os.Exit(2)
	}

	if *showVersion {
		fmt.Println(version)
		return
	}

	args := fs.Args()
	if len(args) != 1 {
		usage()
		os.Exit(2)
	}

	if *imageMode != "download" && *imageMode != "url" {
		fmt.Fprintf(os.Stderr, "error: -image-mode must be 'download' or 'url' (got %q)\n", *imageMode)
		os.Exit(2)
	}

	opts := runner.BoxnoteToMdOptions{
		Input:      args[0],
		OutDir:     *out,
		Recursive:  !*noRecursive,
		Flat:       *flat,
		Overwrite:  *overwrite,
		ImageMode:  *imageMode,
		ImageDir:   *imageDir,
		KeepStyles: *keepStyles,
		DryRun:     *dryRun,
		Verbose:    *verbose,
	}
	summary, err := runner.RunBoxnoteToMd(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if summary.Failed > 0 {
		os.Exit(1)
	}
}
