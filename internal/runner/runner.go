// Package runner は CLI のファイル/ディレクトリ走査と変換実行をまとめる。
package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nanakanok/boxnote2md-cli/internal/boxnote"
	"github.com/nanakanok/boxnote2md-cli/internal/mdparse"
	"github.com/nanakanok/boxnote2md-cli/internal/render"
)

// BoxnoteToMdOptions は boxnote2md コマンドのオプション。
type BoxnoteToMdOptions struct {
	Input      string
	OutDir     string
	Recursive  bool
	Flat       bool
	Overwrite  bool
	ImageMode  string
	ImageDir   string
	KeepStyles bool
	DryRun     bool
	Verbose    bool
}

// MdToBoxnoteOptions は md2boxnote コマンドのオプション。
type MdToBoxnoteOptions struct {
	Input     string
	OutDir    string
	Recursive bool
	Flat      bool
	Overwrite bool
	DryRun    bool
	Verbose   bool
}

// Summary は変換結果のサマリ。
type Summary struct {
	Success  int
	Skipped  int
	Failed   int
	Failures []FailureRecord
}

// FailureRecord は失敗 1 件分の記録。
type FailureRecord struct {
	Path string
	Err  error
}

// RunBoxnoteToMd は .boxnote → .md 変換を実行する。
func RunBoxnoteToMd(opts BoxnoteToMdOptions) (Summary, error) {
	imageDir := opts.ImageDir
	if imageDir == "" {
		imageDir = filepath.Join(opts.OutDir, "images")
	}

	files, root, err := collectInputs(opts.Input, ".boxnote", opts.Recursive)
	if err != nil {
		return Summary{Failed: 1, Failures: []FailureRecord{{Path: opts.Input, Err: err}}}, nil
	}
	if opts.Verbose {
		logf("found %d file(s)", len(files))
	}

	var summary Summary
	taken := map[string]struct{}{}

	for _, src := range files {
		dest := deriveOutPath(src, root, opts.OutDir, ".md", opts.Flat)
		dest = resolveCollision(dest, taken)
		taken[dest] = struct{}{}

		if opts.Verbose {
			logf("converting %s", src)
		}

		existed := fileExists(dest)
		err := convertBoxnoteToMd(src, dest, imageDir, opts)
		if err != nil {
			summary.Failed++
			summary.Failures = append(summary.Failures, FailureRecord{Path: src, Err: err})
			logf("FAILED: %s: %v", src, err)
			continue
		}
		if existed && !opts.Overwrite {
			summary.Skipped++
		} else {
			summary.Success++
		}
	}

	printSummary(summary)
	return summary, nil
}

func convertBoxnoteToMd(src, dest, imageDir string, opts BoxnoteToMdOptions) error {
	if fileExists(dest) && !opts.Overwrite {
		if opts.Verbose || opts.DryRun {
			logf("  -> %s [skipped]", dest)
		}
		return nil
	}
	doc, err := boxnote.Read(src)
	if err != nil {
		return err
	}
	ctx := &render.Context{
		KeepStyles: opts.KeepStyles,
		ImageMode:  opts.ImageMode,
		ImageDir:   imageDir,
		MdPath:     dest,
	}
	md, err := render.Document(doc, ctx)
	if err != nil {
		return err
	}
	if opts.Verbose {
		for _, w := range ctx.Warnings {
			logf("  warn: %s", w)
		}
	}
	if opts.DryRun {
		logf("  -> %s [would-write]", dest)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dest, []byte(md), 0o644); err != nil {
		return err
	}
	if opts.Verbose {
		logf("  -> %s [written]", dest)
	}
	return nil
}

// RunMdToBoxnote は .md → .boxnote 変換を実行する。
func RunMdToBoxnote(opts MdToBoxnoteOptions) (Summary, error) {
	files, root, err := collectInputs(opts.Input, ".md", opts.Recursive)
	if err != nil {
		return Summary{Failed: 1, Failures: []FailureRecord{{Path: opts.Input, Err: err}}}, nil
	}
	if opts.Verbose {
		logf("found %d file(s)", len(files))
	}

	var summary Summary
	taken := map[string]struct{}{}
	for _, src := range files {
		dest := deriveOutPath(src, root, opts.OutDir, ".boxnote", opts.Flat)
		dest = resolveCollision(dest, taken)
		taken[dest] = struct{}{}

		if opts.Verbose {
			logf("converting %s", src)
		}

		existed := fileExists(dest)
		err := convertMdToBoxnote(src, dest, opts)
		if err != nil {
			summary.Failed++
			summary.Failures = append(summary.Failures, FailureRecord{Path: src, Err: err})
			logf("FAILED: %s: %v", src, err)
			continue
		}
		if existed && !opts.Overwrite {
			summary.Skipped++
		} else {
			summary.Success++
		}
	}

	printSummary(summary)
	return summary, nil
}

func convertMdToBoxnote(src, dest string, opts MdToBoxnoteOptions) error {
	if fileExists(dest) && !opts.Overwrite {
		if opts.Verbose || opts.DryRun {
			logf("  -> %s [skipped]", dest)
		}
		return nil
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	doc := mdparse.Parse(string(raw))
	env := mdparse.WrapEnvelope(doc)

	if opts.DryRun {
		logf("  -> %s [would-write]", dest)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := json.Marshal(env)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dest, out, 0o644); err != nil {
		return err
	}
	if opts.Verbose {
		logf("  -> %s [written]", dest)
	}
	return nil
}

// ============================================================
// 共通ユーティリティ
// ============================================================

func collectInputs(input, ext string, recursive bool) ([]string, string, error) {
	info, err := os.Stat(input)
	if err != nil {
		return nil, "", fmt.Errorf("input not found or unsupported: %s: %w", input, err)
	}
	if !info.IsDir() {
		return []string{input}, input, nil
	}
	var files []string
	if recursive {
		err = filepath.WalkDir(input, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.EqualFold(filepath.Ext(path), ext) {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, "", err
		}
	} else {
		entries, err := os.ReadDir(input)
		if err != nil {
			return nil, "", err
		}
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ext) {
				files = append(files, filepath.Join(input, e.Name()))
			}
		}
	}
	sort.Strings(files)
	return files, input, nil
}

func deriveOutPath(src, root, outDir, newExt string, flat bool) string {
	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)) + newExt
	if flat || src == root {
		return filepath.Join(outDir, base)
	}
	rel, err := filepath.Rel(root, src)
	if err != nil {
		return filepath.Join(outDir, base)
	}
	rel = strings.TrimSuffix(rel, filepath.Ext(rel)) + newExt
	return filepath.Join(outDir, rel)
}

func resolveCollision(path string, taken map[string]struct{}) string {
	if _, exists := taken[path]; !exists {
		return path
	}
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(filepath.Base(path), ext)
	for i := 1; ; i++ {
		cand := filepath.Join(dir, fmt.Sprintf("%s-%d%s", stem, i, ext))
		if _, exists := taken[cand]; !exists {
			return cand
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func printSummary(s Summary) {
	logf("summary: success=%d skipped=%d failed=%d", s.Success, s.Skipped, s.Failed)
	if len(s.Failures) > 0 {
		logf("failures:")
		for _, f := range s.Failures {
			logf("  - %s: %v", f.Path, f.Err)
		}
	}
}

// ensure errors.New is used so the package compiles even when error path changes.
var _ = errors.New
