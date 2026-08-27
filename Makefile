GO ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || cat VERSION 2>/dev/null || echo dev)
SOURCE ?= $(shell git config --get remote.origin.url 2>/dev/null || echo https://github.com/jormar/md2confluence.git)
LDFLAGS := -s -w -X 'main.version=$(VERSION)' -X 'main.source=$(SOURCE)'

.PHONY: help test lint build install clean tidy

help:
	@echo "make test     Tests (race + coverage)"
	@echo "make lint     golangci-lint (Docker, falls verfügbar)"
	@echo "make build    Baut bin/md2c $(VERSION)"
	@echo "make install  Installiert nach $$HOME/.local/bin/md2c"
	@echo "make tidy     go mod tidy"
	@echo
	@echo "Zugang: ~/.config/md2c/md2c.conf  (siehe md2c.conf.example)"

test:
	$(GO) test -v -race -cover ./...

lint:
	@if docker info >/dev/null 2>&1; then \
		docker run --rm -v "$(CURDIR):/src" -w /src golangci/golangci-lint:v1.64.8 golangci-lint run; \
	else \
		echo "make lint braucht Docker" >&2; \
		exit 1; \
	fi

build:
	mkdir -p bin
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/md2c ./cmd/md2c

install: build
	mkdir -p "$(HOME)/.local/bin"
	install -m 0755 bin/md2c "$(HOME)/.local/bin/md2c"
	@echo "installed $(HOME)/.local/bin/md2c"

tidy:
	$(GO) mod tidy

clean:
	rm -f bin/md2c bin/md2c.exe coverage.out
