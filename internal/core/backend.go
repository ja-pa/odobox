package core

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	stdmail "net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

const (
	defaultDBPath      = "./voicemail.db"
	defaultCfgPath     = "./config.ini"
	tesseractBinary    = "/usr/bin/tesseract"
	pdftoppmBinary     = "/usr/bin/pdftoppm"
	defaultOCRLanguage = "ces+eng"
)

type appConfig struct {
	sections map[string]map[string]string
}

type imapConfig struct {
	Host     string
	Port     int
	SSL      bool
	Username string
	Password string
	Folder   string
}

type cleanerConfig struct {
	KeepLineRegex      string
	RemoveRegexes      []string
	CollapseBlankLines bool
	VersionV1Regex     string
	VersionV2Regex     string
}

type parserConfig struct {
	CallerPhoneRegex string
}

type smsParserConfig struct {
	TextExtractRegex string
}

type smsConfig struct {
	User      string
	Password  string
	DefaultID string
}

type ListVoicemailsRequest struct {
	Days    int    `json:"days"`
	Clean   bool   `json:"clean"`
	Checked string `json:"checked"`
	Version string `json:"version"`
}

type VoicemailItem struct {
	ID            int          `json:"id"`
	DateReceived  string       `json:"date_received"`
	Subject       string       `json:"subject"`
	CallerPhone   *string      `json:"caller_phone"`
	MessageText   string       `json:"message_text"`
	IsChecked     bool         `json:"is_checked"`
	Attachment    string       `json:"attachment_name"`
	MP3Downloaded bool         `json:"mp3_downloaded"`
	AudioSeconds  *int         `json:"audio_duration_s"`
	Contact       *ContactInfo `json:"contact,omitempty"`
}

type ListVoicemailsResponse struct {
	Items   []VoicemailItem `json:"items"`
	Count   int             `json:"count"`
	Clean   bool            `json:"clean"`
	Version string          `json:"version"`
}

type ListSMSMessagesRequest struct {
	Days    int    `json:"days"`
	Checked string `json:"checked"`
}

type SMSMessageItem struct {
	ID             int          `json:"id"`
	DateReceived   string       `json:"date_received"`
	Subject        string       `json:"subject"`
	SenderPhone    *string      `json:"sender_phone"`
	MessageText    string       `json:"message_text"`
	AttachmentText string       `json:"attachment_text"`
	IsChecked      bool         `json:"is_checked"`
	Attachment     string       `json:"attachment_name"`
	PDFDownloaded  bool         `json:"pdf_downloaded"`
	Contact        *ContactInfo `json:"contact,omitempty"`
}

type ListSMSMessagesResponse struct {
	Items []SMSMessageItem `json:"items"`
	Count int              `json:"count"`
}

type SyncResponse struct {
	Status            string `json:"status"`
	Days              int    `json:"days"`
	Stored            int    `json:"stored"`
	SkippedDuplicates int    `json:"skipped_duplicates"`
	VoicemailStored   int    `json:"voicemail_stored"`
	SMSStored         int    `json:"sms_stored"`
	VoicemailSkipped  int    `json:"voicemail_skipped"`
	SMSSkipped        int    `json:"sms_skipped"`
}

type IMAPDebugItem struct {
	Seq         uint32 `json:"seq"`
	UID         uint32 `json:"uid"`
	Date        string `json:"date"`
	From        string `json:"from"`
	Subject     string `json:"subject"`
	SMSLike     bool   `json:"sms_like"`
	Voicemail   bool   `json:"voicemail_like"`
	HasFromHost bool   `json:"has_odorik_host"`
}

type IMAPPartDebug struct {
	Kind        string `json:"kind"`
	ContentType string `json:"content_type"`
	Filename    string `json:"filename"`
	SizeBytes   int    `json:"size_bytes"`
	Sample      string `json:"sample"`
}

type IMAPMessageDebug struct {
	Seq              uint32          `json:"seq"`
	UID              uint32          `json:"uid"`
	From             string          `json:"from"`
	Subject          string          `json:"subject"`
	Date             string          `json:"date"`
	Parts            []IMAPPartDebug `json:"parts"`
	SMSPDFCount      int             `json:"sms_pdf_count"`
	SMSInlineTextLen int             `json:"sms_inline_text_len"`
}

type UpdateCheckedResponse struct {
	Status    string `json:"status"`
	ID        int    `json:"id"`
	IsChecked bool   `json:"is_checked"`
}

type SettingsResponse struct {
	Settings         map[string]map[string]any `json:"settings"`
	EditableSections []string                  `json:"editable_sections"`
}

type PatchSettingsRequest struct {
	Settings map[string]map[string]any `json:"settings"`
}

type PatchSettingsResponse struct {
	Status   string                    `json:"status"`
	Settings map[string]map[string]any `json:"settings"`
}

type SendSMSRequest struct {
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
	Sender    string `json:"sender"`
}

type SendSMSResponse struct {
	Status           string `json:"status"`
	Recipient        string `json:"recipient"`
	Sender           string `json:"sender"`
	Encoding         string `json:"encoding"`
	CharsUsed        int    `json:"chars_used"`
	MaxSingleChars   int    `json:"max_single_chars"`
	ProviderResponse string `json:"provider_response"`
	SentAt           string `json:"sent_at"`
}

type OdorikBalanceResponse struct {
	Status           string `json:"status"`
	Balance          string `json:"balance"`
	Currency         string `json:"currency"`
	ProviderResponse string `json:"provider_response"`
	UpdatedAt        string `json:"updated_at"`
}

type ContactInfo struct {
	ID        int    `json:"id"`
	FullName  string `json:"full_name"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Org       string `json:"org"`
	Note      string `json:"note"`
	VCard     string `json:"vcard"`
	UpdatedAt string `json:"updated_at"`
}

type ImportVCFRequest struct {
	Content string `json:"content"`
}

type ImportVCFResponse struct {
	Status    string `json:"status"`
	Imported  int    `json:"imported"`
	Updated   int    `json:"updated"`
	Skipped   int    `json:"skipped"`
	Processed int    `json:"processed"`
}

type ExportVCFResponse struct {
	Status  string `json:"status"`
	Content string `json:"content"`
	Count   int    `json:"count"`
}

type CreateContactRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Org      string `json:"org"`
	Note     string `json:"note"`
}

type UpdateContactRequest struct {
	ID       int    `json:"id"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Org      string `json:"org"`
	Note     string `json:"note"`
}

type DeleteContactResponse struct {
	Status string `json:"status"`
	ID     int    `json:"id"`
}

type SMSTemplate struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CreateSMSTemplateRequest struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

type UpdateSMSTemplateRequest struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Body string `json:"body"`
}

type DeleteSMSTemplateResponse struct {
	Status string `json:"status"`
	ID     int    `json:"id"`
}

type Backend struct {
	configPath string
}

func NewBackend(configPath string) *Backend {
	return &Backend{configPath: configPath}
}

func normalizeVersion(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "v1":
		return "v1"
	case "2", "v2":
		return "v2"
	case "both", "all":
		return "all"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func cleanMessageText(text string, cfg cleanerConfig) string {
	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	keepRe := regexp.MustCompile(cfg.KeepLineRegex)
	lines := strings.Split(normalized, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if keepRe.MatchString(line) {
			kept = append(kept, strings.TrimSpace(line))
		}
	}
	if len(kept) > 0 {
		return strings.Join(kept, "\n")
	}
	cleaned := normalized
	for _, pattern := range cfg.RemoveRegexes {
		re := regexp.MustCompile(ensureMultilinePattern(pattern))
		cleaned = re.ReplaceAllString(cleaned, "")
	}
	cleaned = stripOdorikFooterNoise(cleaned)
	if cfg.CollapseBlankLines {
		re := regexp.MustCompile(`\n{3,}`)
		cleaned = re.ReplaceAllString(cleaned, "\n\n")
	}
	return strings.TrimSpace(cleaned)
}

func cleanupWithRegexes(text string, cfg cleanerConfig) string {
	cleaned := text
	for _, pattern := range cfg.RemoveRegexes {
		re := regexp.MustCompile(ensureMultilinePattern(pattern))
		cleaned = re.ReplaceAllString(cleaned, "")
	}
	cleaned = stripOdorikFooterNoise(cleaned)
	if cfg.CollapseBlankLines {
		re := regexp.MustCompile(`\n{3,}`)
		cleaned = re.ReplaceAllString(cleaned, "\n\n")
	}
	return strings.TrimSpace(cleaned)
}

func ensureMultilinePattern(pattern string) string {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "(?m)") || strings.HasPrefix(trimmed, "(?im)") || strings.HasPrefix(trimmed, "(?mi)") {
		return trimmed
	}
	return "(?m)" + trimmed
}

func stripOdorikFooterNoise(text string) string {
	cleaned := text
	footerLine := regexp.MustCompile(`(?mi)^Více informací o přepisu nahrávky na text:.*$`)
	cleaned = footerLine.ReplaceAllString(cleaned, "")
	urlLine := regexp.MustCompile(`(?m)^https?://\S+$`)
	cleaned = urlLine.ReplaceAllString(cleaned, "")
	return cleaned
}

func extractVersion(text string, cfg cleanerConfig, version string) string {
	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	if version == "all" {
		parts := []string{}
		if v1 := extractVersion(normalized, cfg, "v1"); v1 != "" {
			parts = append(parts, v1)
		}
		if v2 := extractVersion(normalized, cfg, "v2"); v2 != "" {
			parts = append(parts, v2)
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
		return cleanMessageText(normalized, cfg)
	}
	if version != "v1" && version != "v2" {
		return cleanMessageText(normalized, cfg)
	}
	pattern := cfg.normalizePattern(cfg.VersionV1Regex)
	if version == "v2" {
		pattern = cfg.normalizePattern(cfg.VersionV2Regex)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		if extracted, ok := extractVersionByMarkers(normalized, cfg, version); ok {
			return extracted
		}
		return ""
	}
	m := re.FindStringSubmatch(normalized)
	if m == nil {
		if extracted, ok := extractVersionByMarkers(normalized, cfg, version); ok {
			return extracted
		}
		return ""
	}
	idx := re.SubexpIndex("content")
	if idx < 0 {
		if len(m) < 2 {
			return ""
		}
		idx = 1
	}
	content := cleanupWithRegexes(strings.TrimSpace(m[idx]), cfg)
	if content == "" {
		return ""
	}
	return version + ": " + content
}

func extractVersionByMarkers(text string, cfg cleanerConfig, version string) (string, bool) {
	targetPrefix := version + ":"
	lines := strings.Split(text, "\n")
	collecting := false
	var out []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lowered := strings.ToLower(trimmed)
		if strings.HasPrefix(lowered, targetPrefix) {
			collecting = true
			content := strings.TrimSpace(trimmed[len(targetPrefix):])
			if content != "" {
				out = append(out, content)
			}
			continue
		}
		if strings.HasPrefix(lowered, "v1:") || strings.HasPrefix(lowered, "v2:") {
			if collecting {
				break
			}
			continue
		}
		if collecting {
			out = append(out, line)
		}
	}

	if len(out) == 0 {
		return "", false
	}
	content := cleanupWithRegexes(strings.TrimSpace(strings.Join(out, "\n")), cfg)
	if content == "" {
		return "", false
	}
	return version + ": " + content, true
}

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

func parseDateHeader(value string) *time.Time {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := stdmail.ParseDate(value)
	if err != nil {
		return nil
	}
	utc := parsed.UTC().Truncate(time.Second)
	return &utc
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func normalizeRecipient(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "(", "")
	s = strings.ReplaceAll(s, ")", "")
	return s
}

func normalizePhoneLoose(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	re := regexp.MustCompile(`\D+`)
	s = re.ReplaceAllString(s, "")
	if strings.HasPrefix(s, "00") {
		s = strings.TrimPrefix(s, "00")
	}
	// Canonicalize Czech numbers so +420/00420 and local 9-digit form match.
	if strings.HasPrefix(s, "420") && len(s) == 12 {
		s = s[3:]
	}
	return s
}

func validateRecipient(raw string) (string, error) {
	recipient := normalizeRecipient(raw)
	if recipient == "" {
		return "", fmt.Errorf("recipient is required")
	}
	if strings.HasPrefix(recipient, "+") {
		recipient = "00" + strings.TrimPrefix(recipient, "+")
	}
	// Allow Czech local and country-prefixed forms entered without "00".
	if reLocalCZ := regexp.MustCompile(`^\d{9}$`); reLocalCZ.MatchString(recipient) {
		recipient = "00420" + recipient
	} else if reCZNoPrefix := regexp.MustCompile(`^420\d{9}$`); reCZNoPrefix.MatchString(recipient) {
		recipient = "00" + recipient
	}
	valid := regexp.MustCompile(`^00\d{6,15}$`)
	if !valid.MatchString(recipient) {
		return "", fmt.Errorf("recipient must be in international format starting with 00 (e.g. 00420123456789)")
	}
	return recipient, nil
}

func validateContact(fullName, phone string) (string, string, error) {
	name := strings.TrimSpace(fullName)
	phoneDisplay := strings.TrimSpace(phone)
	if name == "" {
		return "", "", fmt.Errorf("full_name is required")
	}
	if phoneDisplay == "" {
		return "", "", fmt.Errorf("phone is required")
	}
	normalized := normalizePhoneLoose(phoneDisplay)
	if len(normalized) < 6 || len(normalized) > 15 {
		return "", "", fmt.Errorf("phone must contain 6 to 15 digits")
	}
	return name, phoneDisplay, nil
}

func gsmCharUnits(message string) (int, bool) {
	// GSM-7 basic + extension table (extension chars consume 2 units).
	const gsmBasic = "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞ !\"#¤%&'()*+,-./0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà"
	const gsmExt = "^{}\\[~]|€"
	basic := map[rune]bool{}
	ext := map[rune]bool{}
	for _, r := range gsmBasic {
		basic[r] = true
	}
	for _, r := range gsmExt {
		ext[r] = true
	}
	units := 0
	for _, r := range message {
		if basic[r] {
			units++
			continue
		}
		if ext[r] {
			units += 2
			continue
		}
		return 0, false
	}
	return units, true
}

func singleSMSSegmentInfo(message string) (encoding string, used int, limit int) {
	if units, ok := gsmCharUnits(message); ok {
		return "GSM-7", units, 160
	}
	return "UCS-2", utf8.RuneCountInString(message), 70
}

func composeSMSMessage(identityText, body string) string {
	identity := strings.TrimSpace(identityText)
	message := strings.TrimSpace(body)
	if identity == "" {
		return message
	}
	if message == "" {
		return identity
	}
	return fmt.Sprintf("%s: %s", identity, message)
}

func (b *Backend) logSMS(
	db *sql.DB,
	recipient string,
	sender string,
	message string,
	encoding string,
	charsUsed int,
	maxChars int,
	providerResponse string,
	success bool,
	errMsg string,
) {
	_, _ = db.Exec(
		`INSERT INTO sms_outbox
		(created_at, recipient, sender_id, message_text, encoding, chars_used, max_chars, provider_response, success, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		time.Now().UTC().Format("2006-01-02 15:04:05"),
		recipient,
		sender,
		message,
		encoding,
		charsUsed,
		maxChars,
		providerResponse,
		boolToInt(success),
		errMsg,
	)
}

func (b *Backend) sendSMS(req SendSMSRequest) (SendSMSResponse, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return SendSMSResponse{}, err
	}
	smsCfg := b.loadSMSConfig(cfg)
	if smsCfg.User == "" || smsCfg.Password == "" {
		return SendSMSResponse{}, fmt.Errorf("missing Odorik SMS credentials in settings (set odorik.user and odorik.password; account_id/api_pin also supported)")
	}

	recipient, err := validateRecipient(req.Recipient)
	if err != nil {
		return SendSMSResponse{}, err
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return SendSMSResponse{}, fmt.Errorf("message is required")
	}
	message = composeSMSMessage(cfg.get("app", "sms_identity_text", ""), message)

	encoding, used, limit := singleSMSSegmentInfo(message)
	if used > limit {
		return SendSMSResponse{}, fmt.Errorf(
			"message is too long for one SMS (%s %d/%d). Shorten text so it fits a single segment",
			encoding,
			used,
			limit,
		)
	}

	sender := strings.TrimSpace(req.Sender)
	if sender == "" {
		sender = smsCfg.DefaultID
	}

	form := url.Values{}
	form.Set("user", smsCfg.User)
	form.Set("password", smsCfg.Password)
	form.Set("recipient", recipient)
	form.Set("message", message)
	if sender != "" {
		form.Set("sender", sender)
	}
	form.Set("user_agent", "OdoBox-Wails")

	httpClient := &http.Client{Timeout: 35 * time.Second}
	resp, err := postFormWithRetry(httpClient, "https://www.odorik.cz/api/v1/sms", form, 3)
	if err != nil {
		if isTimeoutError(err) {
			return SendSMSResponse{}, fmt.Errorf("sms request failed: network timeout while connecting to Odorik (try again)")
		}
		return SendSMSResponse{}, fmt.Errorf("sms request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	provider := strings.TrimSpace(string(body))
	if provider == "" {
		provider = "empty_response"
	}

	db, dbErr := openDB(b.resolveDBPath(cfg))
	if dbErr == nil {
		defer db.Close()
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if dbErr == nil {
			b.logSMS(db, recipient, sender, message, encoding, used, limit, provider, false, resp.Status)
		}
		return SendSMSResponse{}, fmt.Errorf("sms provider rejected request: %s (%s)", resp.Status, provider)
	}
	if strings.HasPrefix(strings.ToLower(provider), "error") || strings.HasPrefix(strings.ToLower(provider), "bad_") {
		if dbErr == nil {
			b.logSMS(db, recipient, sender, message, encoding, used, limit, provider, false, provider)
		}
		if strings.Contains(strings.ToLower(provider), "authentication_failed") {
			return SendSMSResponse{}, fmt.Errorf(
				"sms provider error: %s. Verify Odorik Account ID and API password at https://www.odorik.cz/ucet/nastaveni_uctu.html?ucet_podmenu=api_heslo",
				provider,
			)
		}
		return SendSMSResponse{}, fmt.Errorf("sms provider error: %s", provider)
	}

	if dbErr == nil {
		b.logSMS(db, recipient, sender, message, encoding, used, limit, provider, true, "")
	}

	return SendSMSResponse{
		Status:           "ok",
		Recipient:        recipient,
		Sender:           sender,
		Encoding:         encoding,
		CharsUsed:        used,
		MaxSingleChars:   limit,
		ProviderResponse: provider,
		SentAt:           time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func extractBalanceValue(raw string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return ""
	}
	re := regexp.MustCompile(`[-+]?\d+(?:[.,]\d+)?`)
	value := re.FindString(clean)
	if value == "" {
		return clean
	}
	return strings.ReplaceAll(value, ",", ".")
}

func (b *Backend) getOdorikBalance() (OdorikBalanceResponse, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return OdorikBalanceResponse{}, err
	}
	smsCfg := b.loadSMSConfig(cfg)
	if smsCfg.User == "" || smsCfg.Password == "" {
		return OdorikBalanceResponse{}, fmt.Errorf("missing Odorik credentials in settings")
	}

	form := url.Values{}
	form.Set("user", smsCfg.User)
	form.Set("password", smsCfg.Password)
	form.Set("user_agent", "OdoBox-Wails")

	httpClient := &http.Client{Timeout: 20 * time.Second}
	endpoint := "https://www.odorik.cz/api/v1/balance?" + form.Encode()
	var resp *http.Response
	var reqErr error
	for i := 0; i < 3; i++ {
		req, buildErr := http.NewRequest(http.MethodGet, endpoint, nil)
		if buildErr != nil {
			return OdorikBalanceResponse{}, fmt.Errorf("balance request failed: %w", buildErr)
		}
		resp, reqErr = httpClient.Do(req)
		if reqErr == nil {
			break
		}
		if !isTimeoutError(reqErr) || i == 2 {
			break
		}
		time.Sleep(time.Duration(i+1) * 700 * time.Millisecond)
	}
	if reqErr != nil {
		if isTimeoutError(reqErr) {
			return OdorikBalanceResponse{}, fmt.Errorf("balance request failed: network timeout while connecting to Odorik")
		}
		return OdorikBalanceResponse{}, fmt.Errorf("balance request failed: %w", reqErr)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	provider := strings.TrimSpace(string(body))
	if provider == "" {
		provider = "empty_response"
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return OdorikBalanceResponse{}, fmt.Errorf("balance provider rejected request: %s (%s)", resp.Status, provider)
	}
	lowerProvider := strings.ToLower(provider)
	if strings.HasPrefix(lowerProvider, "error") || strings.HasPrefix(lowerProvider, "bad_") {
		if strings.Contains(lowerProvider, "authentication_failed") {
			return OdorikBalanceResponse{}, fmt.Errorf(
				"balance provider error: %s. Verify Odorik Account ID and API password at https://www.odorik.cz/ucet/nastaveni_uctu.html?ucet_podmenu=api_heslo",
				provider,
			)
		}
		return OdorikBalanceResponse{}, fmt.Errorf("balance provider error: %s", provider)
	}

	currency := "Kc"
	if strings.Contains(lowerProvider, "eur") {
		currency = "EUR"
	}
	return OdorikBalanceResponse{
		Status:           "ok",
		Balance:          extractBalanceValue(provider),
		Currency:         currency,
		ProviderResponse: provider,
		UpdatedAt:        time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func postFormWithRetry(client *http.Client, endpoint string, form url.Values, attempts int) (*http.Response, error) {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		resp, err := client.PostForm(endpoint, form)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !isTimeoutError(err) || i == attempts-1 {
			return nil, err
		}
		time.Sleep(time.Duration(i+1) * 700 * time.Millisecond)
	}
	return nil, lastErr
}

func isTimeoutError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "tls handshake timeout") || strings.Contains(lower, "timeout")
}

func validateTemplateFields(name, body string) (string, string, error) {
	cleanName := strings.TrimSpace(name)
	cleanBody := strings.TrimSpace(body)
	if cleanName == "" {
		return "", "", fmt.Errorf("template name is required")
	}
	if cleanBody == "" {
		return "", "", fmt.Errorf("template body is required")
	}
	if utf8.RuneCountInString(cleanName) > 120 {
		return "", "", fmt.Errorf("template name is too long (max 120 chars)")
	}
	if utf8.RuneCountInString(cleanBody) > 2000 {
		return "", "", fmt.Errorf("template body is too long (max 2000 chars)")
	}
	return cleanName, cleanBody, nil
}

func (b *Backend) listSMSTemplates() ([]SMSTemplate, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return nil, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, name, body, created_at, updated_at FROM sms_templates ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []SMSTemplate{}
	for rows.Next() {
		var t SMSTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Body, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (b *Backend) createSMSTemplate(req CreateSMSTemplateRequest) (SMSTemplate, error) {
	name, body, err := validateTemplateFields(req.Name, req.Body)
	if err != nil {
		return SMSTemplate{}, err
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return SMSTemplate{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return SMSTemplate{}, err
	}
	defer db.Close()

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	res, err := db.Exec(`INSERT INTO sms_templates(name, body, created_at, updated_at) VALUES(?, ?, ?, ?)`, name, body, now, now)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return SMSTemplate{}, fmt.Errorf("template name already exists")
		}
		return SMSTemplate{}, err
	}
	id64, _ := res.LastInsertId()
	return SMSTemplate{ID: int(id64), Name: name, Body: body, CreatedAt: now, UpdatedAt: now}, nil
}

func (b *Backend) updateSMSTemplate(req UpdateSMSTemplateRequest) (SMSTemplate, error) {
	if req.ID < 1 {
		return SMSTemplate{}, fmt.Errorf("template id must be positive")
	}
	name, body, err := validateTemplateFields(req.Name, req.Body)
	if err != nil {
		return SMSTemplate{}, err
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return SMSTemplate{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return SMSTemplate{}, err
	}
	defer db.Close()

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	res, err := db.Exec(`UPDATE sms_templates SET name = ?, body = ?, updated_at = ? WHERE id = ?`, name, body, now, req.ID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return SMSTemplate{}, fmt.Errorf("template name already exists")
		}
		return SMSTemplate{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return SMSTemplate{}, fmt.Errorf("template not found")
	}

	var createdAt string
	if err := db.QueryRow(`SELECT created_at FROM sms_templates WHERE id = ?`, req.ID).Scan(&createdAt); err != nil {
		return SMSTemplate{}, err
	}
	return SMSTemplate{ID: req.ID, Name: name, Body: body, CreatedAt: createdAt, UpdatedAt: now}, nil
}

func (b *Backend) deleteSMSTemplate(id int) (DeleteSMSTemplateResponse, error) {
	if id < 1 {
		return DeleteSMSTemplateResponse{}, fmt.Errorf("template id must be positive")
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return DeleteSMSTemplateResponse{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return DeleteSMSTemplateResponse{}, err
	}
	defer db.Close()

	res, err := db.Exec(`DELETE FROM sms_templates WHERE id = ?`, id)
	if err != nil {
		return DeleteSMSTemplateResponse{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return DeleteSMSTemplateResponse{}, fmt.Errorf("template not found")
	}
	return DeleteSMSTemplateResponse{Status: "ok", ID: id}, nil
}

type parsedVCard struct {
	FullName string
	Phones   []string
	Email    string
	Org      string
	Note     string
	Raw      string
}

func unfoldVCardLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && len(out) > 0 {
			out[len(out)-1] += strings.TrimLeft(line, " \t")
			continue
		}
		out = append(out, line)
	}
	return out
}

func unescapeVCardValue(v string) string {
	repl := strings.NewReplacer(`\n`, "\n", `\N`, "\n", `\,`, ",", `\;`, ";", `\\`, `\`)
	return strings.TrimSpace(repl.Replace(v))
}

func parseVCFContacts(content string) []parsedVCard {
	normalized := strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	lines = unfoldVCardLines(lines)

	contacts := []parsedVCard{}
	var current *parsedVCard
	var rawLines []string

	flush := func() {
		if current == nil {
			return
		}
		current.Raw = strings.TrimSpace(strings.Join(rawLines, "\n"))
		if current.FullName != "" && len(current.Phones) > 0 {
			contacts = append(contacts, *current)
		}
		current = nil
		rawLines = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "BEGIN:VCARD") {
			flush()
			current = &parsedVCard{}
			rawLines = append(rawLines, line)
			continue
		}
		if current == nil {
			continue
		}
		rawLines = append(rawLines, line)
		if strings.EqualFold(trimmed, "END:VCARD") {
			flush()
			continue
		}
		keyPart, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(strings.Split(keyPart, ";")[0]))
		value = unescapeVCardValue(value)
		switch key {
		case "FN":
			if current.FullName == "" {
				current.FullName = value
			}
		case "N":
			if current.FullName == "" {
				current.FullName = strings.TrimSpace(strings.ReplaceAll(value, ";", " "))
			}
		case "TEL":
			if value != "" {
				current.Phones = append(current.Phones, value)
			}
		case "EMAIL":
			if current.Email == "" {
				current.Email = value
			}
		case "ORG":
			if current.Org == "" {
				current.Org = value
			}
		case "NOTE":
			if current.Note == "" {
				current.Note = value
			}
		}
	}
	flush()
	return contacts
}

func (b *Backend) listContacts() ([]ContactInfo, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return nil, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, full_name, phone_display, COALESCE(email, ''), COALESCE(org, ''), COALESCE(note, ''), COALESCE(vcard, ''), updated_at FROM contacts ORDER BY full_name, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ContactInfo{}
	for rows.Next() {
		var c ContactInfo
		if err := rows.Scan(&c.ID, &c.FullName, &c.Phone, &c.Email, &c.Org, &c.Note, &c.VCard, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (b *Backend) createContact(req CreateContactRequest) (ContactInfo, error) {
	name, phoneDisplay, err := validateContact(req.FullName, req.Phone)
	if err != nil {
		return ContactInfo{}, err
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return ContactInfo{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return ContactInfo{}, err
	}
	defer db.Close()

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	normalized := normalizePhoneLoose(phoneDisplay)
	res, err := db.Exec(
		`INSERT INTO contacts(full_name, phone_display, phone_normalized, email, org, note, vcard, source, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'manual', ?, ?)`,
		name, phoneDisplay, normalized, strings.TrimSpace(req.Email), strings.TrimSpace(req.Org), strings.TrimSpace(req.Note), "", now, now,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return ContactInfo{}, fmt.Errorf("contact with this phone already exists")
		}
		return ContactInfo{}, err
	}
	id64, _ := res.LastInsertId()
	return ContactInfo{
		ID:        int(id64),
		FullName:  name,
		Phone:     phoneDisplay,
		Email:     strings.TrimSpace(req.Email),
		Org:       strings.TrimSpace(req.Org),
		Note:      strings.TrimSpace(req.Note),
		VCard:     "",
		UpdatedAt: now,
	}, nil
}

func (b *Backend) updateContact(req UpdateContactRequest) (ContactInfo, error) {
	if req.ID < 1 {
		return ContactInfo{}, fmt.Errorf("contact id must be positive")
	}
	name, phoneDisplay, err := validateContact(req.FullName, req.Phone)
	if err != nil {
		return ContactInfo{}, err
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return ContactInfo{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return ContactInfo{}, err
	}
	defer db.Close()

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	normalized := normalizePhoneLoose(phoneDisplay)
	res, err := db.Exec(
		`UPDATE contacts SET full_name=?, phone_display=?, phone_normalized=?, email=?, org=?, note=?, source='manual', updated_at=? WHERE id=?`,
		name, phoneDisplay, normalized, strings.TrimSpace(req.Email), strings.TrimSpace(req.Org), strings.TrimSpace(req.Note), now, req.ID,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return ContactInfo{}, fmt.Errorf("contact with this phone already exists")
		}
		return ContactInfo{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ContactInfo{}, fmt.Errorf("contact not found")
	}
	var vcard string
	_ = db.QueryRow(`SELECT COALESCE(vcard, '') FROM contacts WHERE id = ?`, req.ID).Scan(&vcard)
	return ContactInfo{
		ID:        req.ID,
		FullName:  name,
		Phone:     phoneDisplay,
		Email:     strings.TrimSpace(req.Email),
		Org:       strings.TrimSpace(req.Org),
		Note:      strings.TrimSpace(req.Note),
		VCard:     vcard,
		UpdatedAt: now,
	}, nil
}

func (b *Backend) deleteContact(id int) (DeleteContactResponse, error) {
	if id < 1 {
		return DeleteContactResponse{}, fmt.Errorf("contact id must be positive")
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return DeleteContactResponse{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return DeleteContactResponse{}, err
	}
	defer db.Close()

	res, err := db.Exec(`DELETE FROM contacts WHERE id = ?`, id)
	if err != nil {
		return DeleteContactResponse{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return DeleteContactResponse{}, fmt.Errorf("contact not found")
	}
	return DeleteContactResponse{Status: "ok", ID: id}, nil
}

func (b *Backend) importVCF(req ImportVCFRequest) (ImportVCFResponse, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return ImportVCFResponse{}, fmt.Errorf("vcf content is required")
	}
	entries := parseVCFContacts(content)
	if len(entries) == 0 {
		return ImportVCFResponse{}, fmt.Errorf("no valid contacts found in VCF")
	}

	cfg, err := b.loadConfig()
	if err != nil {
		return ImportVCFResponse{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return ImportVCFResponse{}, err
	}
	defer db.Close()

	imported := 0
	updated := 0
	skipped := 0
	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	for _, entry := range entries {
		if entry.FullName == "" || len(entry.Phones) == 0 {
			skipped++
			continue
		}
		for _, phone := range entry.Phones {
			phoneDisplay := strings.TrimSpace(phone)
			normalized := normalizePhoneLoose(phoneDisplay)
			if len(normalized) < 6 {
				skipped++
				continue
			}
			var existingID int
			err := db.QueryRow(`SELECT id FROM contacts WHERE phone_normalized = ?`, normalized).Scan(&existingID)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return ImportVCFResponse{}, err
			}
			if errors.Is(err, sql.ErrNoRows) {
				_, insErr := db.Exec(
					`INSERT INTO contacts(full_name, phone_display, phone_normalized, email, org, note, vcard, source, created_at, updated_at)
					 VALUES (?, ?, ?, ?, ?, ?, ?, 'vcf', ?, ?)`,
					entry.FullName, phoneDisplay, normalized, entry.Email, entry.Org, entry.Note, entry.Raw, now, now,
				)
				if insErr != nil {
					skipped++
					continue
				}
				imported++
				continue
			}
			_, updErr := db.Exec(
				`UPDATE contacts SET full_name=?, phone_display=?, email=?, org=?, note=?, vcard=?, source='vcf', updated_at=? WHERE id=?`,
				entry.FullName, phoneDisplay, entry.Email, entry.Org, entry.Note, entry.Raw, now, existingID,
			)
			if updErr != nil {
				skipped++
				continue
			}
			updated++
		}
	}

	return ImportVCFResponse{
		Status:    "ok",
		Imported:  imported,
		Updated:   updated,
		Skipped:   skipped,
		Processed: len(entries),
	}, nil
}

func escapeVCardValue(v string) string {
	repl := strings.NewReplacer(`\`, `\\`, "\n", `\n`, ",", `\,`, ";", `\;`)
	return repl.Replace(strings.TrimSpace(v))
}

func (b *Backend) exportVCF() (ExportVCFResponse, error) {
	contacts, err := b.listContacts()
	if err != nil {
		return ExportVCFResponse{}, err
	}
	var out strings.Builder
	for _, c := range contacts {
		if strings.TrimSpace(c.FullName) == "" || strings.TrimSpace(c.Phone) == "" {
			continue
		}
		out.WriteString("BEGIN:VCARD\r\n")
		out.WriteString("VERSION:3.0\r\n")
		out.WriteString("FN:")
		out.WriteString(escapeVCardValue(c.FullName))
		out.WriteString("\r\n")
		out.WriteString("TEL;TYPE=CELL:")
		out.WriteString(escapeVCardValue(c.Phone))
		out.WriteString("\r\n")
		if strings.TrimSpace(c.Email) != "" {
			out.WriteString("EMAIL:")
			out.WriteString(escapeVCardValue(c.Email))
			out.WriteString("\r\n")
		}
		if strings.TrimSpace(c.Org) != "" {
			out.WriteString("ORG:")
			out.WriteString(escapeVCardValue(c.Org))
			out.WriteString("\r\n")
		}
		if strings.TrimSpace(c.Note) != "" {
			out.WriteString("NOTE:")
			out.WriteString(escapeVCardValue(c.Note))
			out.WriteString("\r\n")
		}
		out.WriteString("END:VCARD\r\n")
	}
	return ExportVCFResponse{Status: "ok", Content: out.String(), Count: len(contacts)}, nil
}
