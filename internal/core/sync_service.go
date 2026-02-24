package core

import (
	"fmt"
	"strings"
	"time"
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

	if b.mailGatewayFactory == nil {
		return SyncResponse{}, fmt.Errorf("mail gateway not configured")
	}
	if b.syncStoreFactory == nil {
		return SyncResponse{}, fmt.Errorf("sync store not configured")
	}
	if b.ocrService == nil {
		return SyncResponse{}, fmt.Errorf("ocr service not configured")
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
		return SyncResponse{}, err
	}
	defer func() { _ = gateway.Close() }()

	store, err := b.syncStoreFactory.Open(b.resolveDBPath(cfg))
	if err != nil {
		return SyncResponse{}, err
	}
	defer func() { _ = store.Close() }()

	vmStored, vmSkipped, err := syncVoicemailInbox(gateway, store, days)
	if err != nil {
		return SyncResponse{}, err
	}
	smsStored, smsSkipped, err := syncSMSInbox(gateway, store, b.ocrService, days)
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

func syncVoicemailInbox(gateway MailGateway, store SyncStore, days int) (int, int, error) {
	uids, err := gateway.Search(MailSearchCriteria{
		Since: time.Now().AddDate(0, 0, -days),
		From:  "voicemail@odorik.cz",
	})
	if err != nil {
		return 0, 0, err
	}
	if len(uids) == 0 {
		return 0, 0, nil
	}

	messages, err := gateway.FetchBodies(uids, false)
	if err != nil {
		return 0, 0, err
	}

	stored := 0
	skipped := 0
	seenUIDs := make([]uint32, 0, len(messages))

	for _, msg := range messages {
		if len(msg.Raw) == 0 {
			continue
		}
		parsed, parseErr := parseEmail(msg.Raw)
		if parseErr != nil {
			continue
		}
		messageID := strings.TrimSpace(parsed.MessageID)
		if messageID == "" {
			messageID = fmt.Sprintf("uid-%d", msg.UID)
		}
		exists, err := store.VoicemailExists(messageID)
		if err == nil && exists {
			skipped++
			seenUIDs = append(seenUIDs, msg.UID)
			continue
		}
		if len(parsed.MP3s) == 0 {
			continue
		}
		attachment := parsed.MP3s[0]
		duration := inferDurationFromText(parsed.Subject, parsed.MessageText)
		if err := store.InsertVoicemail(SyncVoicemailRecord{
			MessageID:      messageID,
			Date:           parsed.Date,
			Subject:        parsed.Subject,
			MessageText:    parsed.MessageText,
			AttachmentName: attachment.Name,
			AttachmentData: attachment.Data,
			AudioDuration:  duration,
		}); err != nil {
			continue
		}
		stored++
		seenUIDs = append(seenUIDs, msg.UID)
	}
	if len(seenUIDs) > 0 {
		if err := gateway.MarkSeen(seenUIDs); err != nil {
			return 0, 0, err
		}
	}
	return stored, skipped, nil
}

func syncSMSInbox(gateway MailGateway, store SyncStore, ocr OCRService, days int) (int, int, error) {
	uids, err := gateway.Search(MailSearchCriteria{Since: time.Now().AddDate(0, 0, -days)})
	if err != nil {
		return 0, 0, err
	}
	if len(uids) == 0 {
		return 0, 0, nil
	}
	candidateUIDs, err := findSMSCandidateUIDs(gateway, uids)
	if err != nil {
		return 0, 0, err
	}
	if len(candidateUIDs) == 0 {
		return 0, 0, nil
	}

	messages, err := gateway.FetchBodies(candidateUIDs, true)
	if err != nil {
		return 0, 0, err
	}

	stored := 0
	skipped := 0
	seenUIDs := make([]uint32, 0, len(messages))

	for _, msg := range messages {
		if len(msg.Raw) == 0 {
			continue
		}
		if !isSMSInboundMessage(msg.Envelope, msg.Raw) {
			continue
		}
		parsed, parseErr := parseSMSEmail(msg.Raw)
		if parseErr != nil {
			continue
		}
		messageID := strings.TrimSpace(parsed.MessageID)
		if messageID == "" {
			messageID = fmt.Sprintf("uid-%d", msg.UID)
		}
		existingID, existingAttachmentText, found, err := store.FindSMSByMessageID(messageID)
		if err == nil && found && strings.TrimSpace(existingAttachmentText) != "" {
			skipped++
			seenUIDs = append(seenUIDs, msg.UID)
			continue
		}
		inlineText := strings.TrimSpace(parsed.MessageText)
		attachmentText := ""
		attachmentName := ""
		var attachmentData []byte
		if len(parsed.PDFs) > 0 {
			attachment := parsed.PDFs[0]
			attachmentName = attachment.Name
			attachmentData = attachment.Data
			ocrRaw, ocrErr := ocr.ExtractPDFText(attachment.Data, defaultOCRLanguage)
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
		if found {
			if err := store.UpdateSMSMissingData(existingID, SyncSMSRecord{
				SenderPhone:    senderPhone,
				MessageText:    inlineText,
				AttachmentText: attachmentText,
				AttachmentName: attachmentName,
				AttachmentData: attachmentData,
			}); err != nil {
				continue
			}
			stored++
			seenUIDs = append(seenUIDs, msg.UID)
			continue
		}
		if err := store.InsertSMS(SyncSMSRecord{
			MessageID:      messageID,
			Date:           parsed.Date,
			Subject:        parsed.Subject,
			SenderPhone:    senderPhone,
			MessageText:    inlineText,
			AttachmentText: attachmentText,
			AttachmentName: attachmentName,
			AttachmentData: attachmentData,
		}); err != nil {
			continue
		}
		stored++
		seenUIDs = append(seenUIDs, msg.UID)
	}
	if len(seenUIDs) > 0 {
		if err := gateway.MarkSeen(seenUIDs); err != nil {
			return 0, 0, err
		}
	}
	return stored, skipped, nil
}

func findSMSCandidateUIDs(gateway MailGateway, uids []uint32) ([]uint32, error) {
	messages, err := gateway.FetchEnvelopes(uids)
	if err != nil {
		return nil, err
	}
	candidates := make([]uint32, 0, len(messages))
	for _, msg := range messages {
		if isSMSLikeEnvelope(msg.Envelope) {
			candidates = append(candidates, msg.UID)
		}
	}
	return candidates, nil
}
