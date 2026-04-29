VERSION := 0.1.0
LDFLAGS := -s -w -X main.version=$(VERSION)
GOFLAGS := -trimpath
DIST    := dist

.PHONY: all build test lint clean install build-all

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
