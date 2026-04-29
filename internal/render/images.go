package render

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nanakanok/boxnote2md-cli/internal/boxnote"
)

var unsafeFileChars = regexp.MustCompile(`[\x00-\x1f/\\]`)

func renderImage(n *boxnote.Node, ctx *Context) string {
	boxLink := n.AttrString("boxSharedLink")
	boxFileID := n.AttrString("boxFileId")
	fileName := n.AttrString("fileName")
	if fileName == "" {
		fileName = "image"
	}
	alt := n.AttrString("alt")
	if alt == "" {
		alt = fileName
	}
	src := n.AttrString("src")

	target := boxLink
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		target = src
	}

	if target == "" {
		ctx.Warn(fmt.Sprintf("image without URL (fileName=%q)", fileName))
		return fmt.Sprintf("![image:%s](#unavailable)", fileName)
	}

	if ctx.ImageMode == "url" || ctx.ImageMode == "" {
		return fmt.Sprintf("![%s](%s)", alt, target)
	}

	// download mode
	dl := ctx.ImageDownloader
	if dl == nil {
		dl = defaultDownloader
	}
	safe := unsafeFileChars.ReplaceAllString(fileName, "_")
	if boxFileID != "" {
		safe = unsafeFileChars.ReplaceAllString(boxFileID, "_") + "__" + safe
	}
	dir := ctx.ImageDir
	if dir == "" {
		dir = "./out/images"
	}
	saved := dl(target, dir, safe, ctx)
	if saved == "" {
		return fmt.Sprintf("![%s](%s)", alt, target)
	}
	rel := relativePath(saved, ctx.MdPath)
	ctx.ImageResults = append(ctx.ImageResults, ImageResult{SrcURL: target, SavedTo: saved})
	return fmt.Sprintf("![%s](%s)", alt, rel)
}

func renderBoxPreview(n *boxnote.Node, ctx *Context) string {
	link := n.AttrString("boxSharedLink")
	fileName := n.AttrString("fileName")
	if fileName == "" {
		fileName = basenameFromURL(link)
	}
	if fileName == "" {
		fileName = "Box file"
	}
	if link == "" {
		return fmt.Sprintf("[Box: %s](#unavailable)", fileName)
	}
	return fmt.Sprintf("[Box: %s](%s)", fileName, link)
}

// defaultDownloader は HTTP で URL を取得し、destDir/fileName に保存する。
// 失敗時は空文字を返し、ctx に warning を追加する。
func defaultDownloader(rawURL, destDir, fileName string, ctx *Context) string {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		ctx.Warn(fmt.Sprintf("mkdir %s: %v", destDir, err))
		return ""
	}
	dest := filepath.Join(destDir, fileName)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		ctx.Warn(fmt.Sprintf("image download failed for %s: %v", rawURL, err))
		return ""
	}
	req.Header.Set("User-Agent", "boxnote2md-cli/0.1")

	resp, err := client.Do(req)
	if err != nil {
		ctx.Warn(fmt.Sprintf("image download failed for %s: %v", rawURL, err))
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		ctx.Warn(fmt.Sprintf("image download HTTP %d for %s", resp.StatusCode, rawURL))
		return ""
	}

	f, err := os.Create(dest)
	if err != nil {
		ctx.Warn(fmt.Sprintf("create %s: %v", dest, err))
		return ""
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		ctx.Warn(fmt.Sprintf("save %s: %v", dest, err))
		return ""
	}
	return dest
}

// relativePath は anchor (出力 .md) からみた target の相対パスを POSIX 形式で返す。
func relativePath(target, anchorMd string) string {
	if anchorMd == "" {
		return filepath.ToSlash(target)
	}
	absT, errT := filepath.Abs(target)
	absA, errA := filepath.Abs(anchorMd)
	if errT != nil || errA != nil {
		return filepath.ToSlash(target)
	}
	rel, err := filepath.Rel(filepath.Dir(absA), absT)
	if err != nil {
		return filepath.ToSlash(target)
	}
	return filepath.ToSlash(rel)
}

func basenameFromURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parts := strings.Split(u.Path, "/")
	return parts[len(parts)-1]
}

// 未使用 import を防ぐための anchor (boxnote の使用を保証)
var _ = boxnote.Node{}
