// Command md2boxnote converts Markdown files to Box Note (.boxnote).
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
	fmt.Fprintf(os.Stderr, `md2boxnote %s — Markdown を Box Note (.boxnote) に変換する CLI

Usage:
  md2boxnote <input> [options]

Arguments:
  <input>            .md ファイル または ディレクトリ

Options:
  -o, -out <dir>     出力先ディレクトリ (default: ./out)
  -no-recursive      ディレクトリ入力時の再帰探索を無効化
  -flat              出力先直下にフラット配置
  -overwrite         既存 .boxnote を上書き (default: スキップ)
  -dry-run           書き込みせず処理予定だけ表示
  -v                 詳細ログ
  -version           バージョン表示
  -h                 ヘルプ

`, version)
}

func main() {
	fs := flag.NewFlagSet("md2boxnote", flag.ContinueOnError)
	fs.Usage = usage
	fs.SetOutput(os.Stderr)

	out := fs.String("out", "./out", "出力先ディレクトリ")
	fs.StringVar(out, "o", "./out", "出力先ディレクトリ (短縮形)")
	noRecursive := fs.Bool("no-recursive", false, "再帰探索を無効化")
	flat := fs.Bool("flat", false, "フラット配置")
	overwrite := fs.Bool("overwrite", false, "既存 .boxnote を上書き")
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

	opts := runner.MdToBoxnoteOptions{
		Input:     args[0],
		OutDir:    *out,
		Recursive: !*noRecursive,
		Flat:      *flat,
		Overwrite: *overwrite,
		DryRun:    *dryRun,
		Verbose:   *verbose,
	}
	summary, err := runner.RunMdToBoxnote(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if summary.Failed > 0 {
		os.Exit(1)
	}
}
