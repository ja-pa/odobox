package core

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	imap "github.com/emersion/go-imap"
	gomail "github.com/emersion/go-message/mail"
)

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
	client, err := openIMAPClient(imapCfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Logout() }()

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
			SMSLike:     isSMSLikeEnvelope(mailEnvelopeFromIMAP(msg.Envelope)),
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

func mailEnvelopeFromIMAP(env *imap.Envelope) MailEnvelope {
	if env == nil {
		return MailEnvelope{}
	}
	out := MailEnvelope{
		Subject: strings.TrimSpace(env.Subject),
		Date:    env.Date,
		From:    make([]string, 0, len(env.From)),
	}
	for _, from := range env.From {
		if from == nil {
			continue
		}
		email := strings.ToLower(strings.TrimSpace(from.MailboxName + "@" + from.HostName))
		if email != "" {
			out.From = append(out.From, email)
		}
	}
	return out
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
	client, err := openIMAPClient(imapCfg)
	if err != nil {
		return IMAPMessageDebug{}, err
	}
	defer func() { _ = client.Logout() }()

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
