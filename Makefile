GO ?= go
VERSION ?= $(shell git describe --tags --match 'v0.*' --always --dirty 2>/dev/null || cat VERSION 2>/dev/null || echo dev)
SOURCE ?= $(shell git config --get remote.origin.url 2>/dev/null || echo https://github.com/dannyneumann/md2c.git)
LDFLAGS := -s -w -X 'main.version=$(VERSION)' -X 'main.source=$(SOURCE)'

PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

.PHONY: help test lint build install dist hooks clean tidy

help:
	@echo "make test     Tests (race + coverage)"
	@echo "make lint     golangci-lint (Docker, falls verfügbar)"
	@echo "make build    Baut bin/md2c $(VERSION)"
	@echo "make dist     Cross-compile nach dist/"
	@echo "make install  Installiert nach $$HOME/.local/bin/md2c"
	@echo "make hooks    Installiert pre-push (Unit-Tests vor git push)"
	@echo "make tidy     go mod tidy"
	@echo
	@echo "Zugang: ~/.config/md2c/md2c.conf  (siehe md2c.conf.example)"

test:
	$(GO) test -v -race -cover ./...

lint:
	@if docker info >/dev/null 2>&1; then \
		docker run --rm -v "$(CURDIR):/src" -w /src golangci/golangci-lint:v2.4.0 golangci-lint run; \
	else \
		echo "make lint braucht Docker" >&2; \
		exit 1; \
	fi

build:
	mkdir -p bin
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/md2c ./cmd/md2c

dist:
	mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		out="dist/md2c_$(VERSION)_$${os}_$${arch}"; \
		echo "$$out"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o "$$out" ./cmd/md2c; \
	done
	cd dist && { sha256sum md2c_$(VERSION)_* 2>/dev/null || shasum -a 256 md2c_$(VERSION)_*; } > SHA256SUMS

install: build
	mkdir -p "$(HOME)/.local/bin"
	install -m 0755 bin/md2c "$(HOME)/.local/bin/md2c"
	@echo "installed $(HOME)/.local/bin/md2c"

hooks:
	mkdir -p .git/hooks
	cp .githooks/pre-push .git/hooks/pre-push
	chmod +x .git/hooks/pre-push scripts/next-version.sh
	@echo "installed .git/hooks/pre-push"

tidy:
	$(GO) mod tidy

clean:
	rm -f bin/md2c bin/md2c.exe coverage.out
	rm -rf dist
