VERSION := 0.1.0
LDFLAGS := -s -w -X main.version=$(VERSION)
GOFLAGS := -trimpath
DIST    := dist

.PHONY: all build test lint clean install build-all package release

# 開発用: ローカル OS/Arch でビルド
build:
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST)/boxnote2md ./cmd/boxnote2md
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST)/md2boxnote ./cmd/md2boxnote

test:
	go test ./... -count=1

lint:
	go vet ./...
	gofmt -l . | tee /tmp/gofmt-out; test ! -s /tmp/gofmt-out

clean:
	rm -rf $(DIST)/

install:
	go install $(GOFLAGS) -ldflags '$(LDFLAGS)' ./cmd/boxnote2md
	go install $(GOFLAGS) -ldflags '$(LDFLAGS)' ./cmd/md2boxnote

# 配布用: 4 ターゲットへクロスビルド
build-all: $(DIST)/linux-amd64 $(DIST)/darwin-amd64 $(DIST)/darwin-arm64 $(DIST)/windows-amd64

$(DIST)/linux-amd64:
	@mkdir -p $@
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $@/boxnote2md ./cmd/boxnote2md
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $@/md2boxnote ./cmd/md2boxnote

$(DIST)/darwin-amd64:
	@mkdir -p $@
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $@/boxnote2md ./cmd/boxnote2md
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $@/md2boxnote ./cmd/md2boxnote

$(DIST)/darwin-arm64:
	@mkdir -p $@
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $@/boxnote2md ./cmd/boxnote2md
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $@/md2boxnote ./cmd/md2boxnote

$(DIST)/windows-amd64:
	@mkdir -p $@
	GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $@/boxnote2md.exe ./cmd/boxnote2md
	GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $@/md2boxnote.exe ./cmd/md2boxnote

# Release 配布物のパッケージング
# Linux/macOS は tar.gz, Windows は zip。各アーカイブに README.md と LICENSE を同梱。
PKG_NAME := boxnote2md-cli
ARCHIVES := \
	$(DIST)/$(PKG_NAME)-v$(VERSION)-linux-amd64.tar.gz \
	$(DIST)/$(PKG_NAME)-v$(VERSION)-darwin-amd64.tar.gz \
	$(DIST)/$(PKG_NAME)-v$(VERSION)-darwin-arm64.tar.gz \
	$(DIST)/$(PKG_NAME)-v$(VERSION)-windows-amd64.zip

package: build-all $(ARCHIVES) $(DIST)/SHA256SUMS

$(DIST)/$(PKG_NAME)-v$(VERSION)-%.tar.gz:
	@cp README.md LICENSE $(DIST)/$* 2>/dev/null || cp README.md $(DIST)/$*
	tar -czf $@ -C $(DIST) $*

$(DIST)/$(PKG_NAME)-v$(VERSION)-windows-amd64.zip:
	@cp README.md LICENSE $(DIST)/windows-amd64 2>/dev/null || cp README.md $(DIST)/windows-amd64
	cd $(DIST) && zip -qr $(notdir $@) windows-amd64

$(DIST)/SHA256SUMS:
	cd $(DIST) && sha256sum $(PKG_NAME)-v$(VERSION)-*.tar.gz $(PKG_NAME)-v$(VERSION)-*.zip > SHA256SUMS

# v$(VERSION) タグを切って GitHub Release を作成 (要 gh + 認証)
release: package
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin v$(VERSION)
	gh release create v$(VERSION) \
		$(DIST)/$(PKG_NAME)-v$(VERSION)-*.tar.gz \
		$(DIST)/$(PKG_NAME)-v$(VERSION)-*.zip \
		$(DIST)/SHA256SUMS \
		--title "v$(VERSION)" \
		--notes-file release-notes.md
