package core

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func inferDurationFromText(texts ...string) *int {
	re := regexp.MustCompile(`(?i)\bdelka\s+(\d+)\s*s\b`)
	for _, text := range texts {
		if text == "" {
			continue
		}
		m := re.FindStringSubmatch(text)
		if len(m) < 2 {
			continue
		}
		v, err := strconv.Atoi(m[1])
		if err == nil {
			return &v
		}
	}
	return nil
}

func extractCallerPhone(subject, pattern string) *string {
	if subject == "" {
		return nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	m := re.FindStringSubmatch(subject)
	if m == nil {
		return nil
	}
	idx := re.SubexpIndex("caller")
	if idx < 0 {
		if len(m) < 2 {
			return nil
		}
		idx = 1
	}
	caller := strings.TrimSpace(m[idx])
	if caller == "" {
		return nil
	}
	return &caller
}

func lookupContactByPhone(db *sql.DB, phone string) (*ContactInfo, error) {
	normalized := normalizePhoneLoose(phone)
	if normalized == "" {
		return nil, nil
	}
	var contact ContactInfo
	err := db.QueryRow(
		`SELECT id, full_name, phone_display, COALESCE(email, ''), COALESCE(org, ''), COALESCE(note, ''), COALESCE(vcard, ''), updated_at
		 FROM contacts WHERE phone_normalized = ?
		 OR phone_normalized = CASE
		     WHEN ? LIKE '420%' AND LENGTH(?) = 12 THEN SUBSTR(?, 4)
		     WHEN LENGTH(?) = 9 THEN '420' || ?
		     ELSE ''
		 END`,
		normalized,
		normalized,
		normalized,
		normalized,
		normalized,
		normalized,
	).Scan(
		&contact.ID,
		&contact.FullName,
		&contact.Phone,
		&contact.Email,
		&contact.Org,
		&contact.Note,
		&contact.VCard,
		&contact.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &contact, nil
}

func (b *Backend) listVoicemails(req ListVoicemailsRequest) (ListVoicemailsResponse, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return ListVoicemailsResponse{}, err
	}
	cleanerCfg, err := b.loadCleanerConfig(cfg)
	if err != nil {
		return ListVoicemailsResponse{}, err
	}
	parserCfg, err := b.loadParserConfig(cfg)
	if err != nil {
		return ListVoicemailsResponse{}, err
	}
	if req.Days <= 0 {
		req.Days = 7
	}

	checkedRaw := strings.ToLower(strings.TrimSpace(req.Checked))
	if checkedRaw == "" {
		checkedRaw = "all"
	}
	if checkedRaw != "all" && checkedRaw != "true" && checkedRaw != "false" && checkedRaw != "1" && checkedRaw != "0" && checkedRaw != "yes" && checkedRaw != "no" {
		return ListVoicemailsResponse{}, fmt.Errorf("checked must be one of: all,true,false")
	}
	checkedFilter := -1
	if checkedRaw != "all" {
		if parseBoolLoose(checkedRaw, false) {
			checkedFilter = 1
		} else {
			checkedFilter = 0
		}
	}

	version := normalizeVersion(req.Version)
	if version == "" {
		def, derr := b.defaultTranscriptVersion(cfg)
		if derr != nil {
			return ListVoicemailsResponse{}, derr
		}
		version = def
	}
	if version != "all" && version != "v1" && version != "v2" {
		return ListVoicemailsResponse{}, fmt.Errorf("version must be one of: all,v1,v2,1,2,both")
	}

	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return ListVoicemailsResponse{}, err
	}
	defer db.Close()

	query := `
SELECT id, date_received, subject, COALESCE(message_text, ''), COALESCE(is_checked, 0), COALESCE(attachment_name, ''), audio_duration,
       CASE WHEN attachment_data IS NOT NULL AND length(attachment_data) > 0 THEN 1 ELSE 0 END
FROM voicemails
WHERE 1=1`
	args := []any{}
	cutoff := time.Now().UTC().Add(-time.Duration(req.Days) * 24 * time.Hour).Format("2006-01-02 15:04:05")
	query += ` AND date_received >= ?`
	args = append(args, cutoff)
	if checkedFilter >= 0 {
		query += ` AND is_checked = ?`
		args = append(args, checkedFilter)
	}
	query += ` ORDER BY date_received DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return ListVoicemailsResponse{}, err
	}
	defer rows.Close()

	items := []VoicemailItem{}
	for rows.Next() {
		var item VoicemailItem
		var checked int
		var duration sql.NullInt64
		var hasMP3 int
		if err := rows.Scan(&item.ID, &item.DateReceived, &item.Subject, &item.MessageText, &checked, &item.Attachment, &duration, &hasMP3); err != nil {
			return ListVoicemailsResponse{}, err
		}
		item.IsChecked = checked != 0
		item.MP3Downloaded = hasMP3 != 0
		item.CallerPhone = extractCallerPhone(item.Subject, parserCfg.CallerPhoneRegex)
		if item.CallerPhone != nil {
			contact, cErr := lookupContactByPhone(db, *item.CallerPhone)
			if cErr != nil {
				return ListVoicemailsResponse{}, cErr
			}
			item.Contact = contact
		}
		if duration.Valid {
			d := int(duration.Int64)
			item.AudioSeconds = &d
		} else {
			item.AudioSeconds = inferDurationFromText(item.Subject, item.MessageText)
		}
		if req.Clean {
			item.MessageText = extractVersion(item.MessageText, cleanerCfg, version)
		}
		items = append(items, item)
	}

	return ListVoicemailsResponse{Items: items, Count: len(items), Clean: req.Clean, Version: version}, nil
}

func (b *Backend) updateChecked(id int, checked bool) (UpdateCheckedResponse, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return UpdateCheckedResponse{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return UpdateCheckedResponse{}, err
	}
	defer db.Close()

	res, err := db.Exec(`UPDATE voicemails SET is_checked = ? WHERE id = ?`, boolToInt(checked), id)
	if err != nil {
		return UpdateCheckedResponse{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return UpdateCheckedResponse{}, fmt.Errorf("voicemail not found")
	}
	return UpdateCheckedResponse{Status: "ok", ID: id, IsChecked: checked}, nil
}

func (b *Backend) listSMSMessages(req ListSMSMessagesRequest) (ListSMSMessagesResponse, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return ListSMSMessagesResponse{}, err
	}
	smsParserCfg, err := b.loadSMSParserConfig(cfg)
	if err != nil {
		return ListSMSMessagesResponse{}, err
	}
	if req.Days <= 0 {
		req.Days = 7
	}

	checkedRaw := strings.ToLower(strings.TrimSpace(req.Checked))
	if checkedRaw == "" {
		checkedRaw = "all"
	}
	if checkedRaw != "all" && checkedRaw != "true" && checkedRaw != "false" && checkedRaw != "1" && checkedRaw != "0" && checkedRaw != "yes" && checkedRaw != "no" {
		return ListSMSMessagesResponse{}, fmt.Errorf("checked must be one of: all,true,false")
	}
	checkedFilter := -1
	if checkedRaw != "all" {
		if parseBoolLoose(checkedRaw, false) {
			checkedFilter = 1
		} else {
			checkedFilter = 0
		}
	}

	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return ListSMSMessagesResponse{}, err
	}
	defer db.Close()

	query := `
SELECT id, date_received, subject, COALESCE(sender_phone, ''), COALESCE(message_text, ''), COALESCE(attachment_text, ''), COALESCE(is_checked, 0), COALESCE(attachment_name, ''),
       CASE WHEN attachment_data IS NOT NULL AND length(attachment_data) > 0 THEN 1 ELSE 0 END
FROM sms_inbox
WHERE 1=1`
	args := []any{}
	cutoff := time.Now().UTC().Add(-time.Duration(req.Days) * 24 * time.Hour).Format("2006-01-02 15:04:05")
	query += ` AND date_received >= ?`
	args = append(args, cutoff)
	if checkedFilter >= 0 {
		query += ` AND is_checked = ?`
		args = append(args, checkedFilter)
	}
	query += ` ORDER BY date_received DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return ListSMSMessagesResponse{}, err
	}
	defer rows.Close()

	items := []SMSMessageItem{}
	for rows.Next() {
		var item SMSMessageItem
		var checked int
		var senderPhoneRaw string
		var hasPDF int
		if err := rows.Scan(&item.ID, &item.DateReceived, &item.Subject, &senderPhoneRaw, &item.MessageText, &item.AttachmentText, &checked, &item.Attachment, &hasPDF); err != nil {
			return ListSMSMessagesResponse{}, err
		}
		item.IsChecked = checked != 0
		item.PDFDownloaded = hasPDF != 0
		sourceText := strings.TrimSpace(item.AttachmentText)
		if sourceText == "" {
			sourceText = strings.TrimSpace(item.MessageText)
		}
		item.MessageText = extractSMSUserMessage(sourceText, smsParserCfg)
		item.AttachmentText = extractSMSUserMessage(strings.TrimSpace(item.AttachmentText), smsParserCfg)
		if strings.TrimSpace(senderPhoneRaw) != "" {
			sender := strings.TrimSpace(senderPhoneRaw)
			item.SenderPhone = &sender
			contact, cErr := lookupContactByPhone(db, sender)
			if cErr != nil {
				return ListSMSMessagesResponse{}, cErr
			}
			item.Contact = contact
		}
		items = append(items, item)
	}

	return ListSMSMessagesResponse{Items: items, Count: len(items)}, nil
}

func (b *Backend) updateSMSChecked(id int, checked bool) (UpdateCheckedResponse, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return UpdateCheckedResponse{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return UpdateCheckedResponse{}, err
	}
	defer db.Close()

	res, err := db.Exec(`UPDATE sms_inbox SET is_checked = ? WHERE id = ?`, boolToInt(checked), id)
	if err != nil {
		return UpdateCheckedResponse{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return UpdateCheckedResponse{}, fmt.Errorf("sms message not found")
	}
	return UpdateCheckedResponse{Status: "ok", ID: id, IsChecked: checked}, nil
}

func (b *Backend) getVoicemailAudioDataURL(id int) (string, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return "", err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return "", err
	}
	defer db.Close()

	var audio []byte
	if err := db.QueryRow(`SELECT attachment_data FROM voicemails WHERE id = ?`, id).Scan(&audio); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("voicemail or MP3 attachment not found")
		}
		return "", err
	}
	if len(audio) == 0 {
		return "", fmt.Errorf("voicemail or MP3 attachment not found")
	}
	return "data:audio/mpeg;base64," + base64.StdEncoding.EncodeToString(audio), nil
}
