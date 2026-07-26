# Mew quality gates. Prefer plain `go` commands from CONTRIBUTING.md on Windows without Make.

GO ?= go
BIN_DIR := bin

.PHONY: test vet lint race fuzz-smoke conformance vuln build allowlist

test:
	$(GO) test ./... -count=1

vet:
	$(GO) vet ./...

lint:
	golangci-lint run ./...

race:
	$(GO) test -race ./... -count=1

fuzz-smoke:
	@if command -v pwsh >/dev/null 2>&1; then pwsh -File tools/fuzz-smoke.ps1; \
	elif command -v powershell >/dev/null 2>&1; then powershell -File tools/fuzz-smoke.ps1; \
	else bash tools/fuzz-smoke.sh; fi

conformance:
	$(GO) test ./tests/conformance/... -count=1

vuln:
	govulncheck ./...

VERSION ?= 0.0.0-dev
COMMIT ?=
BUILD_DATE ?=
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/m$(EXE) ./cmd/m
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/mx$(EXE) ./cmd/mx

allowlist:
	$(GO) run ./tools/check-license
	$(GO) run ./tools/check-deps

# EXE is .exe on Windows when set by the caller; empty elsewhere.
EXE ?=
