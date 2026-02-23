SHELL := /bin/bash

WAILS ?= wails
GO ?= go
NPM ?= npm
TAGS ?= webkit2_41
FRONTEND_DIR := frontend
CONFIG_FILE ?= config.ini
CONFIG_EXAMPLE ?= config.ini.example
OCR_TEST_INPUT ?= ./sample-input.pdf

.PHONY: help deps bindings dev dev-browser dev-frontend backend-check backend-test frontend-build release release-ci cli cli-run ocr ocr-test config-example clean

help:
	@echo "Targets:"
	@echo "  make deps           - Install/update Go and frontend dependencies"
	@echo "  make bindings       - Regenerate Wails JS bindings"
	@echo "  make dev            - Full Wails desktop dev (Linux WebKit 4.1 tag)"
	@echo "  make dev-browser    - Full-stack dev in browser via Wails dev server"
	@echo "  make dev-frontend   - Frontend-only Vite dev server"
	@echo "  make backend-check  - Backend compile check"
	@echo "  make backend-test   - Run backend tests"
	@echo "  make frontend-build - Production frontend build"
	@echo "  make release        - Release desktop build + CLI + config example"
	@echo "  make release-ci     - CI release build for RELEASE_OS/RELEASE_ARCH"
	@echo "  make cli            - Build odobox-cli binary (voicemail + sms inbox)"
	@echo "  make cli-run        - Run odobox-cli (go run, pass ARGS='...')"
	@echo "  make ocr            - Build odobox-ocr utility"
	@echo "  make ocr-test       - Run odobox-ocr on OCR_TEST_INPUT=$(OCR_TEST_INPUT)"
	@echo "  make config-example - Generate sanitized config.ini.example"
	@echo "  make clean          - Remove frontend dist artifacts"

deps:
	$(GO) mod tidy
	cd $(FRONTEND_DIR) && $(NPM) install

bindings:
	$(WAILS) generate module

dev:
	$(WAILS) dev -tags "$(TAGS)"

dev-browser:
	$(WAILS) dev -browser -tags "$(TAGS)"

dev-frontend:
	cd $(FRONTEND_DIR) && $(NPM) run dev

backend-check:
	$(GO) build ./...

backend-test:
	$(GO) test ./...

frontend-build:
	cd $(FRONTEND_DIR) && $(NPM) run build

release: config-example
	$(WAILS) build -tags "$(TAGS)"
	@if [ -d cmd/odobox-cli ]; then \
		$(GO) build -o build/bin/odobox-cli ./cmd/odobox-cli; \
	else \
		$(GO) build -tags "cli" -o build/bin/odobox-cli .; \
	fi
	$(GO) build -o build/bin/odobox-ocr ./cmd/odobox-ocr
	cp -f $(CONFIG_EXAMPLE) build/bin/$(CONFIG_EXAMPLE)

RELEASE_OS ?= linux
RELEASE_ARCH ?= amd64

release-ci: config-example
	$(WAILS) build -clean -platform "$(RELEASE_OS)/$(RELEASE_ARCH)" -tags "$(TAGS)"
	@if [ -d cmd/odobox-cli ]; then \
		$(GO) build -o build/bin/odobox-cli ./cmd/odobox-cli; \
	else \
		$(GO) build -tags "cli" -o build/bin/odobox-cli .; \
	fi
	$(GO) build -o build/bin/odobox-ocr ./cmd/odobox-ocr
	cp -f $(CONFIG_EXAMPLE) build/bin/$(CONFIG_EXAMPLE)

cli:
	@if [ -d cmd/odobox-cli ]; then \
		$(GO) build -o odobox-cli ./cmd/odobox-cli; \
	else \
		$(GO) build -tags "cli" -o odobox-cli .; \
	fi

cli-run:
	@if [ -d cmd/odobox-cli ]; then \
		$(GO) run ./cmd/odobox-cli $(ARGS); \
	else \
		$(GO) run -tags "cli" . $(ARGS); \
	fi

ocr:
	$(GO) build -o odobox-ocr ./cmd/odobox-ocr

ocr-test: ocr
	@test -f "$(OCR_TEST_INPUT)" || (echo "Missing OCR input file: $(OCR_TEST_INPUT)"; exit 1)
	./odobox-ocr -input "$(OCR_TEST_INPUT)" -lang ces+eng -output /tmp/odobox-ocr-output.txt
	@echo "OCR output saved to /tmp/odobox-ocr-output.txt"

config-example:
	@test -f "$(CONFIG_FILE)" || (echo "Missing $(CONFIG_FILE)"; exit 1)
	@awk '\
		/^[[:space:]]*$$/ { print; next } \
		/^[[:space:]]*[#;]/ { print; next } \
		/^[[:space:]]*\[.*\][[:space:]]*$$/ { print; next } \
		/^[[:space:]]*[A-Za-z0-9_.-]+[[:space:]]*=/ { \
			line=$$0; \
			sub(/^[[:space:]]*/, "", line); \
			sub(/[[:space:]]*=.*/, "", line); \
			print line " ="; \
			next; \
		} \
		{ next } \
	' "$(CONFIG_FILE)" > "$(CONFIG_EXAMPLE)"
	@echo "Generated $(CONFIG_EXAMPLE)"

clean:
	rm -rf $(FRONTEND_DIR)/dist
