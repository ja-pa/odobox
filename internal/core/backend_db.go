package core

import (
	"database/sql"
	"strings"
)

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := ensureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func ensureSchema(db *sql.DB) error {
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS voicemails (
    id INTEGER PRIMARY KEY,
    message_id TEXT UNIQUE,
    date_received DATETIME,
    subject TEXT,
    message_text TEXT,
    is_checked INTEGER NOT NULL DEFAULT 0,
    attachment_name TEXT,
    attachment_data BLOB,
    audio_duration INTEGER
)`); err != nil {
		return err
	}
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS sms_outbox (
    id INTEGER PRIMARY KEY,
    created_at DATETIME NOT NULL,
    recipient TEXT NOT NULL,
    sender_id TEXT,
    message_text TEXT NOT NULL,
    encoding TEXT NOT NULL,
    chars_used INTEGER NOT NULL,
    max_chars INTEGER NOT NULL,
    provider_response TEXT,
    success INTEGER NOT NULL DEFAULT 0,
    error_message TEXT
)`); err != nil {
		return err
	}
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS sms_templates (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
)`); err != nil {
		return err
	}
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS contacts (
    id INTEGER PRIMARY KEY,
    full_name TEXT NOT NULL,
    phone_display TEXT NOT NULL,
    phone_normalized TEXT NOT NULL UNIQUE,
    email TEXT,
    org TEXT,
    note TEXT,
    vcard TEXT,
    source TEXT NOT NULL DEFAULT 'manual',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
	)`); err != nil {
		return err
	}
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS sms_inbox (
    id INTEGER PRIMARY KEY,
    message_id TEXT UNIQUE,
    date_received DATETIME,
    subject TEXT,
    sender_phone TEXT,
    message_text TEXT,
    attachment_text TEXT,
    is_checked INTEGER NOT NULL DEFAULT 0,
    attachment_name TEXT,
    attachment_data BLOB
)`); err != nil {
		return err
	}
	rows, err := db.Query(`PRAGMA table_info(voicemails)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return err
		}
		cols[strings.ToLower(name)] = true
	}
	if !cols["message_text"] {
		if _, err := db.Exec(`ALTER TABLE voicemails ADD COLUMN message_text TEXT`); err != nil {
			return err
		}
	}
	if !cols["is_checked"] {
		if _, err := db.Exec(`ALTER TABLE voicemails ADD COLUMN is_checked INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	smsRows, err := db.Query(`PRAGMA table_info(sms_inbox)`)
	if err != nil {
		return err
	}
	defer smsRows.Close()
	smsCols := map[string]bool{}
	for smsRows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt any
		if err := smsRows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return err
		}
		smsCols[strings.ToLower(name)] = true
	}
	if !smsCols["sender_phone"] {
		if _, err := db.Exec(`ALTER TABLE sms_inbox ADD COLUMN sender_phone TEXT`); err != nil {
			return err
		}
	}
	if !smsCols["message_text"] {
		if _, err := db.Exec(`ALTER TABLE sms_inbox ADD COLUMN message_text TEXT`); err != nil {
			return err
		}
	}
	if !smsCols["attachment_text"] {
		if _, err := db.Exec(`ALTER TABLE sms_inbox ADD COLUMN attachment_text TEXT`); err != nil {
			return err
		}
	}
	if !smsCols["is_checked"] {
		if _, err := db.Exec(`ALTER TABLE sms_inbox ADD COLUMN is_checked INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
