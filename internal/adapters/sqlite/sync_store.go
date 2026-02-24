package sqlite

import (
	"database/sql"
	"strings"

	"OdorikCentral/internal/core"
)

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

type syncStore struct {
	db *sql.DB
}

func (f *Factory) Open(path string) (core.SyncStore, error) {
	db, err := core.OpenSQLiteDB(path)
	if err != nil {
		return nil, err
	}
	return &syncStore{db: db}, nil
}

func (s *syncStore) Close() error {
	return s.db.Close()
}

func (s *syncStore) VoicemailExists(messageID string) (bool, error) {
	var exists int
	err := s.db.QueryRow(`SELECT 1 FROM voicemails WHERE message_id = ?`, messageID).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}

func (s *syncStore) InsertVoicemail(rec core.SyncVoicemailRecord) error {
	var dtValue any
	if rec.Date != nil {
		dtValue = rec.Date.Format("2006-01-02 15:04:05")
	}
	_, err := s.db.Exec(`
INSERT INTO voicemails (message_id, date_received, subject, message_text, is_checked, attachment_name, attachment_data, audio_duration)
VALUES (?, ?, ?, ?, 0, ?, ?, ?)
`, rec.MessageID, dtValue, rec.Subject, rec.MessageText, rec.AttachmentName, rec.AttachmentData, nullableInt(rec.AudioDuration))
	return err
}

func (s *syncStore) FindSMSByMessageID(messageID string) (id int, attachmentText string, found bool, err error) {
	err = s.db.QueryRow(`SELECT id, COALESCE(attachment_text, '') FROM sms_inbox WHERE message_id = ?`, messageID).Scan(&id, &attachmentText)
	if err == nil {
		return id, attachmentText, true, nil
	}
	if err == sql.ErrNoRows {
		return 0, "", false, nil
	}
	return 0, "", false, err
}

func (s *syncStore) UpdateSMSMissingData(id int, rec core.SyncSMSRecord) error {
	_, err := s.db.Exec(`
UPDATE sms_inbox
SET sender_phone = ?,
    message_text = ?,
    attachment_text = ?,
    attachment_name = CASE WHEN COALESCE(attachment_name, '') = '' THEN ? ELSE attachment_name END,
    attachment_data = CASE WHEN attachment_data IS NULL OR length(attachment_data) = 0 THEN ? ELSE attachment_data END
WHERE id = ?
`, nullableString(rec.SenderPhone), rec.MessageText, rec.AttachmentText, rec.AttachmentName, rec.AttachmentData, id)
	return err
}

func (s *syncStore) InsertSMS(rec core.SyncSMSRecord) error {
	var dtValue any
	if rec.Date != nil {
		dtValue = rec.Date.Format("2006-01-02 15:04:05")
	}
	_, err := s.db.Exec(`
INSERT INTO sms_inbox (message_id, date_received, subject, sender_phone, message_text, attachment_text, is_checked, attachment_name, attachment_data)
VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)
`, rec.MessageID, dtValue, rec.Subject, nullableString(rec.SenderPhone), rec.MessageText, rec.AttachmentText, rec.AttachmentName, rec.AttachmentData)
	return err
}

func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return strings.TrimSpace(v)
}
