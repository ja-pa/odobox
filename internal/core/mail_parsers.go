package core

import (
	"bytes"
	"errors"
	"io"
	stdmail "net/mail"
	"regexp"
	"strings"
	"time"

	imap "github.com/emersion/go-imap"
	gomail "github.com/emersion/go-message/mail"
)

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
