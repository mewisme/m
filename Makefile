# Mew quality gates. Prefer plain `go` commands from CONTRIBUTING.md on Windows without Make.

GO ?= go
BIN_DIR := bin

.PHONY: test vet lint race fuzz-smoke conformance core-cert core-cert-fast core-cert-security core-cert-crash core-cert-performance vuln build allowlist install-dev uninstall-dev

test:
	$(GO) test ./... -count=1

vet:
	$(GO) vet ./...

lint:
	golangci-lint run ./...

race:
	$(GO) test -race ./... -count=1

fuzz-smoke:
	python3 tools/fuzz_smoke.py

conformance:
	$(GO) test ./tests/conformance/... -count=1

core-cert-fast:
	python3 tools/certification/run_core_cert.py core-cert-fast

core-cert:
	python3 tools/certification/run_core_cert.py core-cert

core-cert-security:
	python3 tools/certification/run_core_cert.py core-cert-security

core-cert-crash:
	python3 tools/certification/run_core_cert.py core-cert-crash

core-cert-performance:
	python3 tools/certification/run_core_cert.py core-cert-performance

vuln:
	govulncheck ./...

VERSION ?= 0.0.0-dev
COMMIT ?=
BUILD_DATE ?=
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)

build:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/m$(EXE) ./cmd/m
	CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/mx$(EXE) ./cmd/mx

install-dev:
	@if [ "$$(uname -s)" = "MINGW"* ] || [ "$$(uname -s)" = "MSYS"* ]; then \
	  pwsh -NoProfile -File scripts/install-dev.ps1; \
	else \
	  ./scripts/install-dev.sh; \
	fi

uninstall-dev:
	@if [ "$$(uname -s)" = "MINGW"* ] || [ "$$(uname -s)" = "MSYS"* ]; then \
	  pwsh -NoProfile -File scripts/uninstall-dev.ps1; \
	else \
	  ./scripts/uninstall-dev.sh; \
	fi

allowlist:
	$(GO) run ./tools/check-license
	$(GO) run ./tools/check-deps

# EXE is .exe on Windows when set by the caller; empty elsewhere.
EXE ?=
