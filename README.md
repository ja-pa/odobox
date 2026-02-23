# OdoBox (Wails)

This repository contains the desktop app OdoBox, built with Wails + React frontend + Go backend.

This README is the operational source of truth for runtime behavior, packaging, and release.
For day-to-day engineering workflows, see `README_dev.md`.

## 1. Project Overview

OdoBox is a desktop client that:

- syncs voicemails from IMAP (`voicemail@odorik.cz` sender filter),
- stores voicemail metadata/audio in SQLite,
- sends one-off SMS via Odorik API,
- supports SMS templates (create/edit/delete/use),
- displays/transforms transcripts in React UI,
- lets users manage settings from desktop UI,
- runs as a native Linux desktop app via Wails.

## 2. Repository Layout

- `main.go`: Wails app bootstrap and options.
- `app.go`: Wails adapter exported to frontend bindings.
- `internal/core/`: core business logic and models.
- `internal/adapters/httpapi/`: optional localhost HTTP debug API adapter.
- `cmd/odobox-cli/`: CLI adapter/entrypoint.
- `cmd/odobox-ocr/`: OCR utility entrypoint.
- `frontend/`: React app.
- `frontend/wailsjs/`: auto-generated JS/TS bindings from Go methods.
- `wails.json`: Wails project configuration.
- `Makefile`: standard dev/release commands.

## 3. Runtime Architecture

### 3.1 Backend (Go)

Backend logic in `internal/core` handles:

- INI config load/write/validation,
- SQLite schema creation/migration and CRUD,
- IMAP sync and duplicate skipping,
- SMS send through Odorik endpoint `https://www.odorik.cz/api/v1/sms`,
- checked/unchecked updates,
- transcript version extraction (`v1`, `v2`, `all`),
- cleaning of transcript noise/footer lines,
- audio retrieval as data URL for frontend player.

### 3.2 Frontend (React)

Frontend in `frontend/src`:

- renders inbox/settings/help/address-book views,
- polls sync on configured interval,
- provides manual sidebar resync action,
- calls Go backend through `wailsjs` bindings (optionally also via localhost HTTP API in debug mode).

### 3.3 Wails Bindings

Frontend uses generated methods from `frontend/wailsjs/go/main/App.js`.

Current bound methods:

- `GetSettings()`
- `PatchSettings(req)`
- `ListVoicemails(req)`
- `SyncVoicemails(days)`
- `SetVoicemailChecked(id, checked)`
- `GetVoicemailAudioDataURL(id)`
- `SendSMS(req)`
- `ListSMSTemplates()`
- `CreateSMSTemplate(req)`
- `UpdateSMSTemplate(req)`
- `DeleteSMSTemplate(id)`
- `ListContacts()`
- `ImportVCF(req)`
- `ExportVCF()`
- `CreateContact(req)`
- `UpdateContact(req)`
- `DeleteContact(id)`

If Go signatures change, regenerate bindings:

```bash
make bindings
```

## 4. System Requirements (Ubuntu)

### 4.1 Confirmed Target

- Ubuntu `24.04.x` (Noble)
- Wails CLI `v2.11.0`

### 4.2 Linux GUI Build Dependencies

Install:

```bash
sudo apt update
sudo apt install -y build-essential pkg-config libgtk-3-dev libglib2.0-dev libwebkit2gtk-4.1-dev libsoup-3.0-dev
```

### 4.3 Important Build Tag on Ubuntu 24.04

Wails must build with WebKit 4.1 tag:

- `webkit2_41`

Without this tag, builds may fail with `webkit2gtk-4.0` pkg-config errors.

## 5. Toolchain and Dependencies

- Go: modern Go toolchain (project currently uses `go 1.24.0` in `go.mod`)
- Node/npm for frontend
- Wails CLI v2

Install project deps:

```bash
make deps
```

This runs:

- `go mod tidy`
- `npm install` in `frontend/`

## 6. Build and Run Workflows

### 6.1 Quick Command Reference

```bash
make help
```

Available targets:

- `make deps`
- `make bindings`
- `make dev`
- `make dev-browser`
- `make dev-frontend`
- `make backend-check`
- `make backend-test`
- `make frontend-build`
- `make release`
- `make cli`
- `make cli-run ARGS="..."`
- `make ocr`
- `make ocr-test`
- `make config-example`
- `make clean`

### 6.2 Full Desktop Dev (recommended default)

Runs full Wails app + frontend watcher:

```bash
make dev
```

Equivalent to:

```bash
wails dev -tags webkit2_41
```

Use this when validating real desktop behavior (window runtime, bindings, full integration).

### 6.3 Full-Stack Browser Debug (Go methods available)

```bash
make dev-browser
```

Equivalent to:

```bash
wails dev -browser -tags webkit2_41
```

Use this when you want browser devtools + real backend method calls.

### 6.4 Frontend-Only Debug

```bash
make dev-frontend
```

Runs only Vite. Use this for pure UI/CSS/layout work.

Note: frontend-only mode does not include Wails desktop runtime behavior.

### 6.5 Backend-Only Checks

Compile check:

```bash
make backend-check
```

Tests:

```bash
make backend-test
```

### 6.6 Optional HTTP API for Debugging

The Go backend can expose a local HTTP API while developing/debugging.

Enable via environment:

```bash
ENABLE_HTTP_API=true HTTP_API_PORT=51731 make dev
```

Or in `config.ini`:

```ini
[http_api]
enabled = true
port = 51731
token =
```

Notes:

- API binds only to `127.0.0.1`.
- If `token` is empty, app startup generates a temporary token and prints it in logs.
- Send token in `X-API-Token` or `Authorization: Bearer <token>`.

Quick checks:

```bash
curl -sS http://127.0.0.1:51731/api/health | jq
curl -sS http://127.0.0.1:51731/api/settings -H "X-API-Token: <token>" | jq
```

### 6.7 Frontend Production Build Check

```bash
make frontend-build
```

### 6.8 CLI (odobox-cli)

Build CLI binary:

```bash
make cli
```

Run via `go run` without creating a binary:

```bash
make cli-run ARGS="help"
```

Main CLI commands:

```bash
./odobox-cli list --days 14
./odobox-cli list-sms --days 14
./odobox-cli fetch --days 3
./odobox-cli paths
```

CLI uses the same config/db resolution and core backend logic as the desktop app.

### 6.9 OCR Utility (odobox-ocr)

Build OCR utility:

```bash
make ocr
```

Quick test on default PDF path:

```bash
make ocr-test
```

Manual run:

```bash
./odobox-ocr -input /home/paja/Downloads/aa/input.pdf -lang ces+eng -output /tmp/ocr.txt
```

Implementation calls `/usr/bin/tesseract` directly. For PDF input it first uses `/usr/bin/pdftoppm` to create PNG pages, then runs OCR page by page.

## 7. Release and Packaging

### 7.1 Build Desktop Release

```bash
make release
```

Equivalent to:

```bash
wails build -tags webkit2_41
```

`make release` also builds `build/bin/odobox-cli` and generates a sanitized `config.ini.example` (then copies it to `build/bin/config.ini.example`).
Wails outputs desktop artifacts in the standard Wails build locations (per `wails.json` and Wails defaults).

### 7.2 Recommended Release Checklist

1. Run `make deps`.
2. Run `make bindings`.
3. Run `make backend-check`.
4. Run `make frontend-build`.
5. Run `make release`.
6. Smoke test app launch, inbox list, manual resync, settings save, audio playback.

## 8. Configuration and Data

### 8.1 Config File Resolution

Backend resolves config in this order:

1. `ODORIK_CONFIG` env var (if set)
2. project default `./config.ini` (from `OdorikCentral` cwd)
3. fallback `../../odorik-backend/config.ini` if present

Important: if `./config.ini` is missing in `OdorikCentral`, app will use the fallback file.  
This can cause confusion when Settings UI appears saved but another file is actually used.

### 8.2 DB File Resolution

Backend resolves DB in this order:

1. `ODORIK_DB` env var (if set)
2. `[app].db` from config
3. local `./voicemail.db`
4. fallback `../../odorik-backend/voicemail.db` if present

### 8.3 Config Sections Used

- `[app]`
- `[imap]`
- `[message_cleaner]`
- `[voicemail_parser]`
- `[sms_parser]`
- `[odorik]`
- `[http_api]` (optional, debug only)

For SMS, `[odorik]` supports:

- `user` (preferred; for your account this is typically numeric user/account ID)
- `password` (preferred; API password from Odorik)
- `account_id` (alias of `user`)
- `api_pin` (alias of `password`)
- `sender_id` (optional default sender ID)
- `pin` (legacy fallback if `password` is empty)

### 8.4 Settings Update Rules

Editable sections:

- `imap`
- `message_cleaner`
- `voicemail_parser`
- `sms_parser`
- `app`
- `odorik`

Restricted keys:

- `app`: `poll_interval_minutes`, `default_transcript_version`, `sms_identity_text`
- `odorik`: `pin`, `user`, `password`, `account_id`, `api_pin`, `sender_id`

All other section/key updates are rejected with backend error.

## 9. Transcript Cleaning Notes

Go regex engine differs from Python regex engine.

Compatibility behavior implemented:

- strict regex validation is applied to `keep_line_regex` + `remove_regexes`,
- `version_v1_regex` / `version_v2_regex` can fail to compile but extraction falls back to marker-based parsing (`v1:` / `v2:` blocks),
- cleaner strips known Odorik footer noise lines,
- `remove_regexes` are applied in multiline mode.

If transcript content looks wrong, validate config in `[message_cleaner]` first.

## 10. Known Linux/Wails Troubleshooting

### 10.1 `webkit2gtk-4.0` pkg-config errors on Ubuntu 24.04

Symptom:

- build fails with missing `webkit2gtk-4.0`.

Fix:

- use `-tags webkit2_41` and ensure `libwebkit2gtk-4.1-dev` installed.

### 10.2 Wails bindings out of date

Symptom:

- frontend build reports missing exports like `ListVoicemails` from `App.js`.

Fix:

```bash
make bindings
```

Then rebuild frontend.

### 10.3 `Save failed: undefined` in UI

Handled in current code by frontend error normalization. If it appears again, inspect Go error returned from `PatchSettings` and frontend mapping in `frontend/src/errorUtils.js`.

### 10.4 Vite port already in use

Wails/Vite will pick next free port automatically. This is usually harmless.

## 11. Frontend Navigation Scope

Calendar tab is intentionally removed from active UI. Current primary tabs:

- Inbox
- SMS
- Address Book
- Help
- Settings

## 12. SMS Rules and Behavior

- SMS sending is done from the left menu tab `SMS`.
- SMS templates are persisted in SQLite table `sms_templates` and can be created/edited/deleted from UI.
- In SMS form, selecting `No template` clears the current template-applied message text.
- SMS form supports manual sender override (`Sender` field). If empty, default sender from settings is used.
- SMS recipient can be entered manually or selected from Address Book via searchable picker.
- Settings include `Default SMS identity text` (e.g. `MUDr. Petra Ticha`), automatically prepended to outgoing SMS body as `Identity: message`.
- App enforces **single-segment SMS only**:
  - GSM-7: max `160` chars (extension chars count as 2)
  - UCS-2: max `70` chars
- Longer messages are blocked in frontend and backend.
- Recipient normalization accepts Czech variants and converts to international format:
  - `+420...` -> `00420...`
  - `420...` -> `00420...`
  - local `9`-digit Czech number -> `00420...`
- Backend logs send attempts in SQLite table `sms_outbox`.

### 12.1 Odorik Auth Verification

If SMS returns `error authentication_failed`, verify credentials directly against Odorik:

```bash
curl -s -X GET "https://www.odorik.cz/api/v1/sms/allowed_sender" \
  -d "user=YOUR_USER_ID" \
  -d "password=YOUR_API_PASSWORD"
```

Expected response is sender list (comma-separated), for example:

`00420...,Odorik.cz,SMSinfo,...`

Then ensure these exact values are stored in app config/settings:

- `[odorik].user`
- `[odorik].password`

Aliases `[odorik].account_id` and `[odorik].api_pin` are supported, but canonical keys are `user/password`.

## 13. Address Book Rules and Behavior

- Address Book data is persisted in SQLite table `contacts`.
- Contacts can be managed manually in Address Book page (create/delete).
- Address Book has live search by name, phone, email, organization, and note.
- Each contact row has `SMS` quick action, which opens SMS page with recipient prefilled.
- VCF import/export is handled from Settings page (`Address Book VCF` section).
- VCF import parses `FN/N`, `TEL`, `EMAIL`, `ORG`, `NOTE` and stores raw VCF in DB.
- Import performs upsert by normalized phone (insert new, update existing on change).
- One DB row is stored per normalized phone number.
- VCF export generates a single `contacts.vcf` file for all stored contacts.

### 13.1 Inbox Contact Reflection

- Inbox caller lookup uses Address Book by normalized caller phone.
- If contact is known, caller name is shown instead of raw number.
- Caller phone remains visible as subtitle in message card.
- Clicking contact name in Inbox opens contact detail modal with edit form (`full_name`, `phone`, `email`, `org`, `note`).
- Inbox contact card includes `SMS` icon quick action that opens SMS page with recipient prefilled.

## 14. Sync UX Behavior

Inbox sync behavior:

- initial sync on app startup,
- periodic sync using configured poll interval,
- manual resync via sidebar `Sync -> Resync now` action.

Sidebar status shows:

- minutes since last successful sync,
- Odorik credit/balance near Sync (loaded from `https://www.odorik.cz/api/v1/balance` using configured Odorik credentials),
- connected/error state,
- active resync progress (`Resyncing...`).

## 15. Source-of-Truth Rule

This README is the authoritative runtime/release guide for this repository.
`README_dev.md` is the authoritative developer workflow guide.

When runtime behavior, commands, tags, release process, or architecture assumptions change, update the relevant README files in the same change set.
