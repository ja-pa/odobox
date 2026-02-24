package core

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

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
	if b.mailGatewayFactory == nil {
		return nil, fmt.Errorf("mail gateway not configured")
	}
	gateway, err := b.mailGatewayFactory.Open(
		imapCfg.Host,
		imapCfg.Port,
		imapCfg.SSL,
		imapCfg.Username,
		imapCfg.Password,
		imapCfg.Folder,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = gateway.Close() }()

	uids, err := gateway.Search(MailSearchCriteria{Since: time.Now().AddDate(0, 0, -days)})
	if err != nil {
		return nil, err
	}
	if len(uids) == 0 {
		return []IMAPDebugItem{}, nil
	}
	if len(uids) > limit {
		uids = uids[len(uids)-limit:]
	}

	messages, err := gateway.FetchEnvelopes(uids)
	if err != nil {
		return nil, err
	}

	out := []IMAPDebugItem{}
	for _, msg := range messages {
		if strings.TrimSpace(msg.Envelope.Subject) == "" && len(msg.Envelope.From) == 0 {
			continue
		}
		fromStr := strings.TrimSpace(firstEnvelopeFrom(msg.Envelope))
		odorikHost := false
		for _, email := range msg.Envelope.From {
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
			Seq:         msg.Seq,
			UID:         msg.UID,
			Date:        dateVal,
			From:        fromStr,
			Subject:     subject,
			SMSLike:     isSMSLikeEnvelope(msg.Envelope),
			Voicemail:   strings.Contains(fromStr, "voicemail@odorik.cz"),
			HasFromHost: odorikHost,
		}
		out = append(out, item)
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
	if b.mailGatewayFactory == nil {
		return IMAPMessageDebug{}, fmt.Errorf("mail gateway not configured")
	}
	gateway, err := b.mailGatewayFactory.Open(
		imapCfg.Host,
		imapCfg.Port,
		imapCfg.SSL,
		imapCfg.Username,
		imapCfg.Password,
		imapCfg.Folder,
	)
	if err != nil {
		return IMAPMessageDebug{}, err
	}
	defer func() { _ = gateway.Close() }()

	messages, err := gateway.FetchBodies([]uint32{uid}, true)
	if err != nil {
		return IMAPMessageDebug{}, err
	}

	var out IMAPMessageDebug
	for _, msg := range messages {
		if msg.UID == 0 {
			continue
		}
		out.Seq = msg.Seq
		out.UID = msg.UID
		if !msg.Envelope.Date.IsZero() {
			out.Date = msg.Envelope.Date.UTC().Format("2006-01-02 15:04:05")
		}
		out.Subject = strings.TrimSpace(msg.Envelope.Subject)
		out.From = strings.TrimSpace(firstEnvelopeFrom(msg.Envelope))
		if len(msg.Raw) == 0 {
			continue
		}
		reader, readMsgErr := gomail.CreateReader(bytes.NewReader(msg.Raw))
		if readMsgErr != nil {
			continue
		}
		smsParsed, smsParseErr := parseSMSEmail(msg.Raw)
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
	if out.UID == 0 {
		return IMAPMessageDebug{}, fmt.Errorf("uid %d not found", uid)
	}
	return out, nil
}

func firstEnvelopeFrom(env MailEnvelope) string {
	if len(env.From) == 0 {
		return ""
	}
	return env.From[0]
}
