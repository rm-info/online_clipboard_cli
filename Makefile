# Makefile for clibo

VERSION ?= v0.1.0-dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
MODULE  := github.com/rm-info/online_clipboard_cli

LDFLAGS := -s -w \
           -X $(MODULE)/internal/version.Version=$(VERSION) \
           -X $(MODULE)/internal/version.Commit=$(COMMIT)

DIST := dist
BIN  := clibo
CMD  := ./cmd/$(BIN)

# Release matrix. Add or remove rows here; `dist` regenerates the lot.
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all help build test vet check dist install clean

all: build

help:
	@echo "Targets:"
	@echo "  build    Build clibo for the host platform into dist/"
	@echo "  test     Run go test on every package"
	@echo "  vet      Run go vet on every package"
	@echo "  check    vet + test (use before pushing)"
	@echo "  dist     Cross-compile release binaries for every entry in PLATFORMS"
	@echo "  install  Copy the host binary into ~/.local/bin"
	@echo "  clean    Remove the dist/ tree"

build:
	@mkdir -p $(DIST)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BIN) $(CMD)
	@echo "→ $(DIST)/$(BIN)"

test:
	go test ./...

vet:
	go vet ./...

check: vet test

dist: clean
	@mkdir -p $(DIST)
	@for platform in $(PLATFORMS); do \
	  os=$${platform%/*}; arch=$${platform#*/}; \
	  out=$(DIST)/$(BIN)-$$os-$$arch; \
	  if [ "$$os" = "windows" ]; then out=$$out.exe; fi; \
	  echo "→ $$out"; \
	  GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o $$out $(CMD) || exit 1; \
	done
	@echo
	@ls -lh $(DIST)/

install: build
	@mkdir -p $(HOME)/.local/bin
	cp $(DIST)/$(BIN) $(HOME)/.local/bin/$(BIN)
	@echo "Installed $(HOME)/.local/bin/$(BIN)"

clean:
	rm -rf $(DIST)
