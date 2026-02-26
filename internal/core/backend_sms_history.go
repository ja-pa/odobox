package core

import (
	"sort"
	"strings"
	"time"
)

func (b *Backend) listSMSHistory(req ListSMSHistoryRequest) (ListSMSHistoryResponse, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return ListSMSHistoryResponse{}, err
	}
	smsParserCfg, err := b.loadSMSParserConfig(cfg)
	if err != nil {
		return ListSMSHistoryResponse{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return ListSMSHistoryResponse{}, err
	}
	defer db.Close()

	days := req.Days
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, -days).UTC().Format("2006-01-02 15:04:05")

	items := []SMSHistoryItem{}

	outboxRows, err := db.Query(`
SELECT id,
       COALESCE(created_at, ''),
       COALESCE(recipient, ''),
       COALESCE(message_text, ''),
       COALESCE(sender_id, ''),
       COALESCE(success, 0),
       COALESCE(provider_response, ''),
       COALESCE(error_message, '')
FROM sms_outbox
WHERE created_at >= ?
ORDER BY created_at DESC
`, cutoff)
	if err != nil {
		return ListSMSHistoryResponse{}, err
	}
	defer outboxRows.Close()

	for outboxRows.Next() {
		var item SMSHistoryItem
		var successInt int
		if err := outboxRows.Scan(
			&item.ID,
			&item.OccurredAt,
			&item.Counterparty,
			&item.MessageText,
			&item.SenderID,
			&successInt,
			&item.ProviderResponse,
			&item.ErrorMessage,
		); err != nil {
			return ListSMSHistoryResponse{}, err
		}
		item.Direction = "sent"
		item.Success = successInt == 1
		items = append(items, item)
	}
	if err := outboxRows.Err(); err != nil {
		return ListSMSHistoryResponse{}, err
	}

	inboxRows, err := db.Query(`
SELECT id,
       COALESCE(date_received, ''),
       COALESCE(sender_phone, ''),
       COALESCE(message_text, ''),
       COALESCE(subject, ''),
       COALESCE(attachment_text, '')
FROM sms_inbox
WHERE date_received >= ?
ORDER BY date_received DESC
`, cutoff)
	if err != nil {
		return ListSMSHistoryResponse{}, err
	}
	defer inboxRows.Close()

	for inboxRows.Next() {
		var item SMSHistoryItem
		var attachmentText string
		if err := inboxRows.Scan(
			&item.ID,
			&item.OccurredAt,
			&item.Counterparty,
			&item.MessageText,
			&item.Subject,
			&attachmentText,
		); err != nil {
			return ListSMSHistoryResponse{}, err
		}
		item.Direction = "received"
		item.Success = true
		sourceText := strings.TrimSpace(attachmentText)
		if sourceText == "" {
			sourceText = strings.TrimSpace(item.MessageText)
		}
		item.MessageText = extractSMSUserMessage(sourceText, smsParserCfg)
		items = append(items, item)
	}
	if err := inboxRows.Err(); err != nil {
		return ListSMSHistoryResponse{}, err
	}

	sort.Slice(items, func(i, j int) bool {
		return historyTime(items[i].OccurredAt).After(historyTime(items[j].OccurredAt))
	})

	return ListSMSHistoryResponse{Items: items, Count: len(items)}, nil
}

func historyTime(raw string) time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts
	}
	ts, err := time.Parse("2006-01-02 15:04:05", value)
	if err == nil {
		return ts
	}
	return time.Time{}
}
