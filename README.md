# OdoBox Developer Guide

This repository currently keeps a single README focused on development, debugging, refactoring, and release workflows.

## 1. Architecture Map

Data flow:

1. Source A: IMAP inbox (voicemail/SMS-to-PDF mails from Odorik)
2. Source B: Odorik HTTP API (SMS send, balance)
3. Transform/parsing/normalization in Go backend
4. Persist to SQLite
5. Expose data/actions to frontend via Wails bindings (optionally via local HTTP debug API)

Current Go structure:

- `main.go`: Wails bootstrap.
- `app.go`: Wails-facing adapter (exports methods for frontend bindings).
- `internal/core`: business logic, use-cases, DTOs.
- `internal/adapters/httpapi`: optional localhost HTTP API adapter for debug/integration tests.
- `internal/adapters/imap`: IMAP gateway adapter.
- `internal/adapters/sqlite`: SQLite store adapter.
- `internal/adapters/ocr`: OCR adapter.
- `cmd/odobox-cli`: CLI adapter for sync/list/debug operations.
- `cmd/odobox-ocr`: OCR utility adapter.

Frontend structure:

- `frontend/src`: React application.
- `frontend/wailsjs`: generated Wails bindings (regenerate after Go signature changes).

## 2. System Requirements

### 2.1 Confirmed Linux target

- Ubuntu `24.04.x` (Noble)
- Wails CLI `v2.11.0`
- Build tag `webkit2_41`

### 2.2 Linux GUI build dependencies

```bash
sudo apt update
sudo apt install -y build-essential pkg-config libgtk-3-dev libglib2.0-dev libwebkit2gtk-4.1-dev libsoup-3.0-dev
```

### 2.3 Windows OCR prerequisites

Install:

- Tesseract OCR: https://tesseract-ocr.github.io/tessdoc/Installation.html
- Xpdf command line tools (`pdftoppm`): https://www.xpdfreader.com/download.html

## 3. Common Commands

Run from `OdorikCentral/`:

```bash
make help
make deps
make bindings
make backend-check
make backend-test
make frontend-build
make dev
make dev-browser
make dev-frontend
make cli
make cli-run ARGS="help"
make ocr
make ocr-test
make demo-db
make release
```

## 4. Local Development Modes

Desktop app (default):

```bash
make dev
```

Browser + backend methods:

```bash
make dev-browser
```

Frontend only:

```bash
make dev-frontend
```

Demo database:

```bash
make demo-db
ODORIK_CONFIG=demo/config.ini make dev
```

Single command:

```bash
make demo-db && ODORIK_CONFIG=demo/config.ini make dev
```

## 5. HTTP Debug API (Optional)

Enable via environment:

```bash
ENABLE_HTTP_API=true HTTP_API_PORT=51731 make dev
```

Optional static token:

```bash
ENABLE_HTTP_API=true HTTP_API_PORT=51731 HTTP_API_TOKEN=dev-token make dev
```

Config alternative (`config.ini`):

```ini
[http_api]
enabled = true
port = 51731
token =
```

Quick checks:

```bash
curl -sS http://127.0.0.1:51731/api/health | jq
curl -sS http://127.0.0.1:51731/api/settings -H "X-API-Token: dev-token" | jq
```

Notes:

- API binds only to `127.0.0.1`.
- If token is empty, a runtime token is generated and printed in logs.

## 6. CLI Debugging Workflows

Build and run:

```bash
make cli
./odobox-cli help
```

IMAP inspection:

```bash
./odobox-cli debug-imap --days 14 --limit 80
./odobox-cli debug-imap-message --seq 12345
```

Sync and list:

```bash
./odobox-cli fetch --days 7
./odobox-cli list --days 14
./odobox-cli list-sms --days 14
./odobox-cli paths
```

## 7. Config and DB Resolution

Config path priority:

1. `ODORIK_CONFIG`
2. `./config.ini`
3. `../../odorik-backend/config.ini`

DB path priority:

1. `ODORIK_DB`
2. `[app].db` from config
3. `./voicemail.db`
4. `../../odorik-backend/voicemail.db`

## 8. Refactor Rules

When changing architecture:

1. Keep behavior unchanged unless explicitly requested.
2. Keep adapters thin (`app.go`, `cmd/*`, `internal/adapters/*`).
3. Keep business logic in `internal/core` until further migration.
4. Run `go build ./...` and `go test ./...` after each major move.
5. Update this README in the same change.

## 9. Wails Bindings

If Wails-exported signatures change:

```bash
make bindings
```

Then validate frontend:

```bash
make frontend-build
```

## 10. Release Checklist

```bash
make deps
make bindings
make backend-check
make frontend-build
make release
```

## 11. Troubleshooting Notes

### 11.1 `webkit2gtk-4.0` pkg-config errors on Ubuntu 24.04

Use `-tags webkit2_41` and ensure `libwebkit2gtk-4.1-dev` is installed.

### 11.2 Wails bindings out of date

Regenerate with:

```bash
make bindings
```

### 11.3 Vite port already in use

Wails/Vite usually picks next free port automatically.
