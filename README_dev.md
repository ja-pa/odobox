# OdoBox Developer Guide

This file is for day-to-day development, debugging, and refactoring workflows.

## 1. Architecture Map

Current Go structure:

- `app.go`: Wails-facing adapter (exports methods for frontend bindings).
- `main.go`: Wails bootstrap.
- `internal/core`: business logic and data models.
- `internal/adapters/httpapi`: optional localhost HTTP API adapter used for debug/integration tests.
- `cmd/odobox-cli`: CLI adapter for sync/list/debug operations.
- `cmd/odobox-ocr`: OCR utility adapter.

Frontend structure:

- `frontend/src`: React application.
- `frontend/wailsjs`: generated Wails bindings (regenerate after Go signature changes).

## 2. Common Commands

Run from `OdorikCentral/`:

```bash
make help
make deps
make backend-check
make backend-test
make frontend-build
make dev
make dev-browser
make dev-frontend
make cli
make cli-run ARGS="help"
make release
```

## 3. Local Development Modes

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

## 4. HTTP Debug API (Optional)

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

## 5. CLI Debugging Workflows

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

## 6. Config and DB Resolution

Config path priority:

1. `ODORIK_CONFIG`
2. `./config.ini`
3. `../../odorik-backend/config.ini`

DB path priority:

1. `ODORIK_DB`
2. `[app].db` from config
3. `./voicemail.db`
4. `../../odorik-backend/voicemail.db`

## 7. Refactor Rules

When changing architecture:

1. Keep behavior unchanged unless explicitly requested.
2. Keep adapters thin (`app.go`, `cmd/*`, `internal/adapters/*`).
3. Put business logic in `internal/core`.
4. Run `go build ./...` and `go test ./...` after each major move.
5. Update both `README.md` and `README_dev.md` in the same change.

## 8. Regenerating Wails Bindings

If Wails-exported signatures change:

```bash
make bindings
```

Then validate frontend:

```bash
make frontend-build
```

## 9. Release Sanity Checklist

```bash
make deps
make bindings
make backend-check
make frontend-build
make release
```
