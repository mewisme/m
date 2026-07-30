# Mew quality gates. Prefer plain `go` commands from CONTRIBUTING.md on Windows without Make.

GO ?= go
BIN_DIR := bin

.PHONY: test vet lint race fuzz-smoke conformance core-cert core-cert-fast core-cert-security core-cert-crash core-cert-performance vuln build allowlist

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

core-cert-fast:
	@if command -v pwsh >/dev/null 2>&1; then pwsh -File tools/certification/run-core-cert.ps1 -Target core-cert-fast; \
	elif command -v powershell >/dev/null 2>&1; then powershell -File tools/certification/run-core-cert.ps1 -Target core-cert-fast; \
	else sh tools/certification/run-core-cert.sh core-cert-fast; fi

core-cert:
	@if command -v pwsh >/dev/null 2>&1; then pwsh -File tools/certification/run-core-cert.ps1 -Target core-cert; \
	elif command -v powershell >/dev/null 2>&1; then powershell -File tools/certification/run-core-cert.ps1 -Target core-cert; \
	else sh tools/certification/run-core-cert.sh core-cert; fi

core-cert-security:
	@if command -v pwsh >/dev/null 2>&1; then pwsh -File tools/certification/run-core-cert.ps1 -Target core-cert-security; \
	elif command -v powershell >/dev/null 2>&1; then powershell -File tools/certification/run-core-cert.ps1 -Target core-cert-security; \
	else sh tools/certification/run-core-cert.sh core-cert-security; fi

core-cert-crash:
	@if command -v pwsh >/dev/null 2>&1; then pwsh -File tools/certification/run-core-cert.ps1 -Target core-cert-crash; \
	elif command -v powershell >/dev/null 2>&1; then powershell -File tools/certification/run-core-cert.ps1 -Target core-cert-crash; \
	else sh tools/certification/run-core-cert.sh core-cert-crash; fi

core-cert-performance:
	@if command -v pwsh >/dev/null 2>&1; then pwsh -File tools/certification/run-core-cert.ps1 -Target core-cert-performance; \
	elif command -v powershell >/dev/null 2>&1; then powershell -File tools/certification/run-core-cert.ps1 -Target core-cert-performance; \
	else sh tools/certification/run-core-cert.sh core-cert-performance; fi

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
