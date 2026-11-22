GOBIN := $(shell go env GOPATH)/bin
GOLANGCI_LINT := $(GOBIN)/golangci-lint
GOLANGCI_LATEST_URL := https://api.github.com/repos/golangci/golangci-lint/releases/latest
GOLANGCI_INSTALL_URL := https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh

.PHONY: test lint lint-install web run

test:
	cd backend && go test -race -coverprofile=coverage.out -covermode=atomic ./internal/...
	@cd backend && go tool cover -func=coverage.out | tail -n 1

lint:
	@if [ ! -x "$(GOLANGCI_LINT)" ]; then \
		echo "golangci-lint not found at $(GOLANGCI_LINT); run 'make lint-install'"; exit 1; \
	fi; \
	installed=$$($(GOLANGCI_LINT) version --short 2>/dev/null | sed 's/^v//'); \
	latest=$$(curl -fsSL --max-time 10 $(GOLANGCI_LATEST_URL) 2>/dev/null | sed -n 's/.*"tag_name": *"v\([^"]*\)".*/\1/p' | head -n 1); \
	if [ -z "$$latest" ]; then \
		echo "warning: could not fetch the latest golangci-lint release (offline?); running $$installed anyway"; \
	elif [ "$$installed" != "$$latest" ]; then \
		echo "golangci-lint $$installed is installed but $$latest is the latest release; run 'make lint-install'"; exit 1; \
	fi; \
	cd backend && $(GOLANGCI_LINT) run ./...

# The script is downloaded to a file first: in `curl | sh` a failed curl still
# exits 0, and dash (Ubuntu's sh) has no pipefail to catch it.
lint-install:
	@tmp=$$(mktemp) || exit 1; \
	if curl -fsSL --max-time 30 -o "$$tmp" $(GOLANGCI_INSTALL_URL); then \
		sh "$$tmp" -b "$(GOBIN)"; status=$$?; \
	else \
		echo "failed to download $(GOLANGCI_INSTALL_URL)"; status=1; \
	fi; \
	rm -f "$$tmp"; exit $$status

# The glob skips dotfiles, so the versioned dist/.gitkeep survives.
web:
	cd frontend && npm ci && npm run build
	rm -rf backend/web/dist/*
	cp -R frontend/dist/. backend/web/dist/

run:
	cd backend && go run .
