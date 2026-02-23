package core

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	stdmail "net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	imap "github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	gomail "github.com/emersion/go-message/mail"
)

func (b *Backend) sync(days int) (SyncResponse, error) {
	if days < 1 {
		return SyncResponse{}, fmt.Errorf("days must be a positive integer")
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return SyncResponse{}, err
	}
	imapCfg, err := b.loadIMAPConfig(cfg)
	if err != nil {
		return SyncResponse{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return SyncResponse{}, err
	}
	defer db.Close()

	address := fmt.Sprintf("%s:%d", imapCfg.Host, imapCfg.Port)
	var client *imapclient.Client
	if imapCfg.SSL {
		client, err = imapclient.DialTLS(address, nil)
	} else {
		client, err = imapclient.Dial(address)
	}
	if err != nil {
		return SyncResponse{}, err
	}
	defer func() { _ = client.Logout() }()

	if err := client.Login(imapCfg.Username, imapCfg.Password); err != nil {
		return SyncResponse{}, err
	}
	if _, err := client.Select(imapCfg.Folder, false); err != nil {
		return SyncResponse{}, err
	}

	vmStored, vmSkipped, err := syncVoicemailInbox(client, db, days)
	if err != nil {
		return SyncResponse{}, err
	}
	smsStored, smsSkipped, err := syncSMSInbox(client, db, days)
	if err != nil {
		return SyncResponse{}, err
	}
	return SyncResponse{
		Status:            "ok",
		Days:              days,
		Stored:            vmStored + smsStored,
		SkippedDuplicates: vmSkipped + smsSkipped,
		VoicemailStored:   vmStored,
		SMSStored:         smsStored,
		VoicemailSkipped:  vmSkipped,
		SMSSkipped:        smsSkipped,
	}, nil
}

func syncVoicemailInbox(client *imapclient.Client, db *sql.DB, days int) (int, int, error) {
	sinceDate := time.Now().AddDate(0, 0, -days)
	criteria := imap.NewSearchCriteria()
	criteria.Header = textprotoMIMEHeader(map[string][]string{"From": {"voicemail@odorik.cz"}})
	criteria.Since = sinceDate

	uids, err := client.Search(criteria)
	if err != nil {
		return 0, 0, err
	}
	if len(uids) == 0 {
		return 0, 0, nil
	}

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchUid, imap.FetchEnvelope, section.FetchItem()}
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	messages := make(chan *imap.Message, 16)
	fetchErr := make(chan error, 1)
	go func() { fetchErr <- client.Fetch(seqset, items, messages) }()

	stored := 0
	skipped := 0
	seenSet := new(imap.SeqSet)

	for msg := range messages {
		if msg == nil {
			continue
		}
		body := msg.GetBody(section)
		if body == nil {
			continue
		}
		rawEmail, readErr := io.ReadAll(body)
		if readErr != nil {
			continue
		}
		parsed, parseErr := parseEmail(rawEmail)
		if parseErr != nil {
			continue
		}
		messageID := strings.TrimSpace(parsed.MessageID)
		if messageID == "" {
			messageID = fmt.Sprintf("uid-%d", msg.Uid)
		}
		var exists int
		if err := db.QueryRow(`SELECT 1 FROM voicemails WHERE message_id = ?`, messageID).Scan(&exists); err == nil {
			skipped++
			seenSet.AddNum(msg.Uid)
			continue
		}
		if len(parsed.MP3s) == 0 {
			continue
		}
		attachment := parsed.MP3s[0]
		duration := inferDurationFromText(parsed.Subject, parsed.MessageText)
		var dtValue any
		if parsed.Date != nil {
			dtValue = parsed.Date.Format("2006-01-02 15:04:05")
		}
		_, err = db.Exec(`
INSERT INTO voicemails (message_id, date_received, subject, message_text, is_checked, attachment_name, attachment_data, audio_duration)
VALUES (?, ?, ?, ?, 0, ?, ?, ?)
`, messageID, dtValue, parsed.Subject, parsed.MessageText, attachment.Name, attachment.Data, nullableInt(duration))
		if err != nil {
			continue
		}
		stored++
		seenSet.AddNum(msg.Uid)
	}
	if err := <-fetchErr; err != nil {
		return 0, 0, err
	}
	if len(seenSet.Set) > 0 {
		if err := client.UidStore(seenSet, imap.FormatFlagsOp(imap.AddFlags, true), []interface{}{imap.SeenFlag}, nil); err != nil {
			return 0, 0, err
		}
	}
	return stored, skipped, nil
}

func syncSMSInbox(client *imapclient.Client, db *sql.DB, days int) (int, int, error) {
	sinceDate := time.Now().AddDate(0, 0, -days)
	criteria := imap.NewSearchCriteria()
	criteria.Since = sinceDate

	uids, err := client.Search(criteria)
	if err != nil {
		return 0, 0, err
	}
	if len(uids) == 0 {
		return 0, 0, nil
	}

	candidateUIDs, err := findSMSCandidateUIDs(client, uids)
	if err != nil {
		return 0, 0, err
	}
	if len(candidateUIDs) == 0 {
		return 0, 0, nil
	}

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchUid, imap.FetchEnvelope, section.FetchItem()}
	seqset := new(imap.SeqSet)
	seqset.AddNum(candidateUIDs...)
	messages := make(chan *imap.Message, 16)
	fetchErr := make(chan error, 1)
	go func() { fetchErr <- client.UidFetch(seqset, items, messages) }()

	stored := 0
	skipped := 0
	seenSet := new(imap.SeqSet)

	for msg := range messages {
		if msg == nil {
			continue
		}
		body := msg.GetBody(section)
		if body == nil {
			continue
		}
		rawEmail, readErr := io.ReadAll(body)
		if readErr != nil {
			continue
		}
		if !isSMSInboundMessage(msg, rawEmail) {
			continue
		}
		parsed, parseErr := parseSMSEmail(rawEmail)
		if parseErr != nil {
			continue
		}
		messageID := strings.TrimSpace(parsed.MessageID)
		if messageID == "" {
			messageID = fmt.Sprintf("uid-%d", msg.Uid)
		}
		existingID := 0
		existingAttachmentText := ""
		if err := db.QueryRow(`SELECT id, COALESCE(attachment_text, '') FROM sms_inbox WHERE message_id = ?`, messageID).Scan(&existingID, &existingAttachmentText); err == nil {
			if strings.TrimSpace(existingAttachmentText) != "" {
				skipped++
				seenSet.AddNum(msg.Uid)
				continue
			}
		}
		inlineText := strings.TrimSpace(parsed.MessageText)
		attachmentText := ""
		attachmentName := ""
		var attachmentData []byte
		if len(parsed.PDFs) > 0 {
			attachment := parsed.PDFs[0]
			attachmentName = attachment.Name
			attachmentData = attachment.Data
			ocrRaw, ocrErr := ocrPDFData(attachment.Data, defaultOCRLanguage)
			if ocrErr == nil && strings.TrimSpace(ocrRaw) != "" {
				attachmentText = strings.TrimSpace(ocrRaw)
			}
		}
		if inlineText == "" && attachmentText == "" && len(attachmentData) == 0 {
			continue
		}
		senderText := attachmentText
		if senderText == "" {
			senderText = inlineText
		}
		senderPhone := extractSMSSenderPhone(parsed.Subject, senderText)
		var dtValue any
		if parsed.Date != nil {
			dtValue = parsed.Date.Format("2006-01-02 15:04:05")
		}
		if existingID > 0 {
			_, err = db.Exec(`
UPDATE sms_inbox
SET sender_phone = ?,
    message_text = ?,
    attachment_text = ?,
    attachment_name = CASE WHEN COALESCE(attachment_name, '') = '' THEN ? ELSE attachment_name END,
    attachment_data = CASE WHEN attachment_data IS NULL OR length(attachment_data) = 0 THEN ? ELSE attachment_data END
WHERE id = ?
`, nullableString(senderPhone), inlineText, attachmentText, attachmentName, attachmentData, existingID)
			if err != nil {
				continue
			}
			stored++
			seenSet.AddNum(msg.Uid)
			continue
		}

		_, err = db.Exec(`
INSERT INTO sms_inbox (message_id, date_received, subject, sender_phone, message_text, attachment_text, is_checked, attachment_name, attachment_data)
VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)
`, messageID, dtValue, parsed.Subject, nullableString(senderPhone), inlineText, attachmentText, attachmentName, attachmentData)
		if err != nil {
			continue
		}
		stored++
		seenSet.AddNum(msg.Uid)
	}
	if err := <-fetchErr; err != nil {
		return 0, 0, err
	}
	if len(seenSet.Set) > 0 {
		if err := client.UidStore(seenSet, imap.FormatFlagsOp(imap.AddFlags, true), []interface{}{imap.SeenFlag}, nil); err != nil {
			return 0, 0, err
		}
	}
	return stored, skipped, nil
}

func findSMSCandidateUIDs(client *imapclient.Client, uids []uint32) ([]uint32, error) {
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	items := []imap.FetchItem{imap.FetchUid, imap.FetchEnvelope}
	messages := make(chan *imap.Message, 64)
	fetchErr := make(chan error, 1)
	go func() { fetchErr <- client.Fetch(seqset, items, messages) }()

	candidates := make([]uint32, 0, len(uids))
	for msg := range messages {
		if msg == nil {
			continue
		}
		if isSMSLikeEnvelope(msg.Envelope) {
			candidates = append(candidates, msg.Uid)
		}
	}
	if err := <-fetchErr; err != nil {
		return nil, err
	}
	return candidates, nil
}

func isSMSLikeEnvelope(env *imap.Envelope) bool {
	if env == nil {
		return false
	}
	subject := strings.ToLower(strings.TrimSpace(env.Subject))
	if strings.Contains(subject, "sms na pevnou linku") || strings.Contains(subject, "sms to fixed line") || strings.Contains(subject, "fax") {
		return true
	}
	for _, from := range env.From {
		if from == nil {
			continue
		}
		email := strings.ToLower(strings.TrimSpace(from.MailboxName + "@" + from.HostName))
		if looksLikeSMSSender(email) {
			return true
		}
		if strings.Contains(email, "odorik.cz") && (strings.Contains(email, "fax") || strings.Contains(email, "sms")) {
			return true
		}
	}
	return false
}

func looksLikeSMSSender(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false
	}
	if email == "fax_to_mail@odorik.cz" || email == "sms_to_mail@odorik.cz" || email == "fax2mail@odorik.cz" {
		return true
	}
	re := regexp.MustCompile(`(?:fax|sms)[-_]?(?:to|2)[-_]?mail@odorik\.cz`)
	return re.MatchString(email)
}

func isSMSInboundMessage(msg *imap.Message, rawEmail []byte) bool {
	if msg != nil && msg.Envelope != nil {
		if isSMSLikeEnvelope(msg.Envelope) {
			return true
		}
		for _, addr := range msg.Envelope.From {
			if addr == nil {
				continue
			}
			email := strings.ToLower(strings.TrimSpace(addr.MailboxName + "@" + addr.HostName))
			if looksLikeSMSSender(email) {
				return true
			}
		}
	}
	parsed, err := stdmail.ReadMessage(bytes.NewReader(rawEmail))
	if err != nil {
		return false
	}
	from := strings.ToLower(strings.TrimSpace(parsed.Header.Get("From")))
	return looksLikeSMSSender(from) || (strings.Contains(from, "odorik.cz") && (strings.Contains(from, "fax") || strings.Contains(from, "sms")))
}

func (b *Backend) debugIMAP(days int, limit int) ([]IMAPDebugItem, error) {
	if days < 1 {
		days = 1
	}
	if limit < 1 {
		limit = 50
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return nil, err
	}
	imapCfg, err := b.loadIMAPConfig(cfg)
	if err != nil {
		return nil, err
	}
	address := fmt.Sprintf("%s:%d", imapCfg.Host, imapCfg.Port)
	var client *imapclient.Client
	if imapCfg.SSL {
		client, err = imapclient.DialTLS(address, nil)
	} else {
		client, err = imapclient.Dial(address)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Logout() }()
	if err := client.Login(imapCfg.Username, imapCfg.Password); err != nil {
		return nil, err
	}
	if _, err := client.Select(imapCfg.Folder, false); err != nil {
		return nil, err
	}

	criteria := imap.NewSearchCriteria()
	criteria.Since = time.Now().AddDate(0, 0, -days)
	uids, err := client.Search(criteria)
	if err != nil {
		return nil, err
	}
	if len(uids) == 0 {
		return []IMAPDebugItem{}, nil
	}
	if len(uids) > limit {
		uids = uids[len(uids)-limit:]
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	items := []imap.FetchItem{imap.FetchUid, imap.FetchEnvelope}
	messages := make(chan *imap.Message, 64)
	fetchErr := make(chan error, 1)
	go func() { fetchErr <- client.Fetch(seqset, items, messages) }()

	out := []IMAPDebugItem{}
	for msg := range messages {
		if msg == nil || msg.Envelope == nil {
			continue
		}
		fromStr := ""
		odorikHost := false
		for _, f := range msg.Envelope.From {
			if f == nil {
				continue
			}
			email := strings.ToLower(strings.TrimSpace(f.MailboxName + "@" + f.HostName))
			if fromStr == "" {
				fromStr = email
			}
			if strings.Contains(email, "odorik.cz") {
				odorikHost = true
			}
		}
		subject := strings.TrimSpace(msg.Envelope.Subject)
		dateVal := ""
		if !msg.Envelope.Date.IsZero() {
			dateVal = msg.Envelope.Date.UTC().Format("2006-01-02 15:04:05")
		}
		item := IMAPDebugItem{
			Seq:         msg.SeqNum,
			UID:         msg.Uid,
			Date:        dateVal,
			From:        fromStr,
			Subject:     subject,
			SMSLike:     isSMSLikeEnvelope(msg.Envelope),
			Voicemail:   strings.Contains(fromStr, "voicemail@odorik.cz"),
			HasFromHost: odorikHost,
		}
		out = append(out, item)
	}
	if err := <-fetchErr; err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UID > out[j].UID })
	return out, nil
}

func (b *Backend) debugIMAPMessage(uid uint32) (IMAPMessageDebug, error) {
	if uid == 0 {
		return IMAPMessageDebug{}, fmt.Errorf("uid must be positive")
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return IMAPMessageDebug{}, err
	}
	imapCfg, err := b.loadIMAPConfig(cfg)
	if err != nil {
		return IMAPMessageDebug{}, err
	}
	address := fmt.Sprintf("%s:%d", imapCfg.Host, imapCfg.Port)
	var client *imapclient.Client
	if imapCfg.SSL {
		client, err = imapclient.DialTLS(address, nil)
	} else {
		client, err = imapclient.Dial(address)
	}
	if err != nil {
		return IMAPMessageDebug{}, err
	}
	defer func() { _ = client.Logout() }()
	if err := client.Login(imapCfg.Username, imapCfg.Password); err != nil {
		return IMAPMessageDebug{}, err
	}
	if _, err := client.Select(imapCfg.Folder, false); err != nil {
		return IMAPMessageDebug{}, err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchUid, imap.FetchEnvelope, section.FetchItem()}
	messages := make(chan *imap.Message, 1)
	fetchErr := make(chan error, 1)
	go func() { fetchErr <- client.Fetch(seqset, items, messages) }()

	var out IMAPMessageDebug
	for msg := range messages {
		if msg == nil {
			continue
		}
		out.Seq = msg.SeqNum
		out.UID = msg.Uid
		if msg.Envelope != nil {
			if !msg.Envelope.Date.IsZero() {
				out.Date = msg.Envelope.Date.UTC().Format("2006-01-02 15:04:05")
			}
			out.Subject = strings.TrimSpace(msg.Envelope.Subject)
			for _, from := range msg.Envelope.From {
				if from == nil {
					continue
				}
				email := strings.ToLower(strings.TrimSpace(from.MailboxName + "@" + from.HostName))
				if out.From == "" {
					out.From = email
				}
			}
		}
		body := msg.GetBody(section)
		if body == nil {
			continue
		}
		rawEmail, readErr := io.ReadAll(body)
		if readErr != nil {
			continue
		}
		reader, readMsgErr := gomail.CreateReader(bytes.NewReader(rawEmail))
		if readMsgErr != nil {
			continue
		}
		smsParsed, smsParseErr := parseSMSEmail(rawEmail)
		if smsParseErr == nil {
			out.SMSPDFCount = len(smsParsed.PDFs)
			out.SMSInlineTextLen = len(strings.TrimSpace(smsParsed.MessageText))
		}
		for {
			part, partErr := reader.NextPart()
			if partErr != nil {
				if errors.Is(partErr, io.EOF) {
					break
				}
				break
			}
			switch h := part.Header.(type) {
			case *gomail.InlineHeader:
				ct, _, _ := h.ContentType()
				payload, _ := io.ReadAll(part.Body)
				sample := strings.TrimSpace(string(payload))
				sample = strings.Join(strings.Fields(sample), " ")
				if len(sample) > 120 {
					sample = sample[:120] + "..."
				}
				out.Parts = append(out.Parts, IMAPPartDebug{
					Kind:        "inline",
					ContentType: strings.TrimSpace(ct),
					SizeBytes:   len(payload),
					Sample:      sample,
				})
			case *gomail.AttachmentHeader:
				ct, _, _ := h.ContentType()
				filename, _ := h.Filename()
				payload, _ := io.ReadAll(part.Body)
				out.Parts = append(out.Parts, IMAPPartDebug{
					Kind:        "attachment",
					ContentType: strings.TrimSpace(ct),
					Filename:    strings.TrimSpace(filename),
					SizeBytes:   len(payload),
				})
			}
		}
	}
	if err := <-fetchErr; err != nil {
		return IMAPMessageDebug{}, err
	}
	if out.UID == 0 {
		return IMAPMessageDebug{}, fmt.Errorf("sequence %d not found", uid)
	}
	return out, nil
}

type parsedMessage struct {
	MessageID   string
	Date        *time.Time
	Subject     string
	MessageText string
	MP3s        []mp3Attachment
}

type mp3Attachment struct {
	Name string
	Data []byte
}

type parsedSMSEmail struct {
	MessageID   string
	Date        *time.Time
	Subject     string
	MessageText string
	PDFs        []pdfAttachment
}

type pdfAttachment struct {
	Name string
	Data []byte
}

func parseEmail(raw []byte) (parsedMessage, error) {
	reader, err := gomail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return parsedMessage{}, err
	}
	header := reader.Header
	messageID, _ := header.MessageID()
	subject, _ := header.Subject()
	dateHeader, _ := header.Text("Date")
	parsed := parsedMessage{MessageID: strings.TrimSpace(messageID), Date: parseDateHeader(dateHeader), Subject: strings.TrimSpace(subject)}

	for {
		part, err := reader.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return parsedMessage{}, err
		}
		switch h := part.Header.(type) {
		case *gomail.InlineHeader:
			contentType, _, _ := h.ContentType()
			body, _ := io.ReadAll(part.Body)
			if strings.EqualFold(contentType, "text/plain") && parsed.MessageText == "" {
				parsed.MessageText = strings.TrimSpace(string(body))
			}
		case *gomail.AttachmentHeader:
			filename, _ := h.Filename()
			contentType, _, _ := h.ContentType()
			if !strings.HasSuffix(strings.ToLower(filename), ".mp3") && !strings.EqualFold(contentType, "audio/mpeg") && !strings.EqualFold(contentType, "audio/mp3") {
				continue
			}
			payload, _ := io.ReadAll(part.Body)
			if len(payload) == 0 {
				continue
			}
			name := filename
			if strings.TrimSpace(name) == "" {
				name = "voice_message.mp3"
			}
			parsed.MP3s = append(parsed.MP3s, mp3Attachment{Name: name, Data: payload})
		}
	}
	return parsed, nil
}

func parseSMSEmail(raw []byte) (parsedSMSEmail, error) {
	reader, err := gomail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return parsedSMSEmail{}, err
	}
	header := reader.Header
	messageID, _ := header.MessageID()
	subject, _ := header.Subject()
	dateHeader, _ := header.Text("Date")
	parsed := parsedSMSEmail{MessageID: strings.TrimSpace(messageID), Date: parseDateHeader(dateHeader), Subject: strings.TrimSpace(subject)}

	for {
		part, err := reader.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return parsedSMSEmail{}, err
		}
		switch h := part.Header.(type) {
		case *gomail.InlineHeader:
			contentType, _, _ := h.ContentType()
			body, _ := io.ReadAll(part.Body)
			if strings.EqualFold(contentType, "text/plain") && parsed.MessageText == "" {
				parsed.MessageText = strings.TrimSpace(string(body))
				continue
			}
			if strings.EqualFold(contentType, "application/pdf") {
				if len(body) == 0 {
					continue
				}
				parsed.PDFs = append(parsed.PDFs, pdfAttachment{Name: "sms_message.pdf", Data: body})
			}
		case *gomail.AttachmentHeader:
			filename, _ := h.Filename()
			contentType, _, _ := h.ContentType()
			if !strings.HasSuffix(strings.ToLower(filename), ".pdf") && !strings.EqualFold(contentType, "application/pdf") {
				continue
			}
			payload, _ := io.ReadAll(part.Body)
			if len(payload) == 0 {
				continue
			}
			name := filename
			if strings.TrimSpace(name) == "" {
				name = "sms_message.pdf"
			}
			parsed.PDFs = append(parsed.PDFs, pdfAttachment{Name: name, Data: payload})
		}
	}
	return parsed, nil
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

func textprotoMIMEHeader(input map[string][]string) map[string][]string {
	return input
}

func extractSMSSenderPhone(subject, text string) string {
	candidates := []string{
		`(?im)\b(?:od|from)\s*:\s*(\+?\d[\d\s]{5,})`,
		`(?im)\bpro\s*/\s*to:\s*\d+\s+od\s*/\s*from:\s*(\+?\d[\d\s]{5,})`,
		`(?i)(\+?\d{9,15})`,
	}
	for _, pattern := range candidates {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			if v := strings.TrimSpace(m[1]); v != "" {
				return v
			}
		}
		if m := re.FindStringSubmatch(subject); len(m) > 1 {
			if v := strings.TrimSpace(m[1]); v != "" {
				return v
			}
		}
	}
	return ""
}

func extractSMSUserMessage(raw string, cfg smsParserConfig) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	re, err := regexp.Compile(cfg.TextExtractRegex)
	if err != nil {
		return text
	}
	matches := re.FindStringSubmatch(text)
	if len(matches) == 0 {
		return text
	}

	content := ""
	if idx := re.SubexpIndex("content"); idx > 0 && idx < len(matches) {
		content = matches[idx]
	}
	if content == "" {
		for i := 1; i < len(matches); i++ {
			if strings.TrimSpace(matches[i]) != "" {
				content = matches[i]
				break
			}
		}
	}
	if content == "" {
		return text
	}
	content = strings.TrimSpace(content)
	content = strings.Trim(content, `"`)
	content = strings.TrimSpace(content)
	if len(content) >= len("message") && strings.EqualFold(content[:len("message")], "message") {
		content = strings.TrimSpace(content[len("message"):])
	}
	if content == "" {
		return text
	}
	return content
}

func ocrPDFData(pdfData []byte, lang string) (string, error) {
	if len(pdfData) == 0 {
		return "", fmt.Errorf("empty PDF data")
	}
	if strings.TrimSpace(lang) == "" {
		lang = defaultOCRLanguage
	}
	if _, err := os.Stat(tesseractBinary); err != nil {
		return "", fmt.Errorf("tesseract not found: %w", err)
	}
	if _, err := os.Stat(pdftoppmBinary); err != nil {
		return "", fmt.Errorf("pdftoppm not found: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "odobox-ocr-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "input.pdf")
	if err := os.WriteFile(pdfPath, pdfData, 0o600); err != nil {
		return "", err
	}

	prefix := filepath.Join(tmpDir, "page")
	conv := exec.Command(pdftoppmBinary, "-r", "300", "-png", pdfPath, prefix)
	convOut, err := conv.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pdftoppm failed: %w (%s)", err, strings.TrimSpace(string(convOut)))
	}

	pages, err := filepath.Glob(filepath.Join(tmpDir, "page-*.png"))
	if err != nil {
		return "", err
	}
	sort.Strings(pages)
	if len(pages) == 0 {
		return "", fmt.Errorf("no PNG pages produced")
	}

	var out strings.Builder
	for i, page := range pages {
		cmd := exec.Command(tesseractBinary, page, "stdout", "-l", lang, "--psm", "6")
		txt, ocrErr := cmd.CombinedOutput()
		if ocrErr != nil {
			return "", fmt.Errorf("tesseract failed on page %d: %w (%s)", i+1, ocrErr, strings.TrimSpace(string(txt)))
		}
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString(strings.TrimSpace(string(txt)))
		out.WriteString("\n")
	}
	return strings.TrimSpace(out.String()), nil
}
