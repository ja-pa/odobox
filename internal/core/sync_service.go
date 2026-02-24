package core

import (
	"fmt"
	"io"
	"strings"
	"time"

	imap "github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
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
	store := newDBStore(db)

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

	vmStored, vmSkipped, err := syncVoicemailInbox(client, store, days)
	if err != nil {
		return SyncResponse{}, err
	}
	smsStored, smsSkipped, err := syncSMSInbox(client, store, days)
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

func syncVoicemailInbox(client *imapclient.Client, store *dbStore, days int) (int, int, error) {
	sinceDate := time.Now().AddDate(0, 0, -days)
	criteria := imap.NewSearchCriteria()
	criteria.Header = map[string][]string{"From": {"voicemail@odorik.cz"}}
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
		exists, err := store.voicemailExists(messageID)
		if err == nil && exists {
			skipped++
			seenSet.AddNum(msg.Uid)
			continue
		}
		if len(parsed.MP3s) == 0 {
			continue
		}
		attachment := parsed.MP3s[0]
		duration := inferDurationFromText(parsed.Subject, parsed.MessageText)
		if err := store.insertVoicemail(voicemailRecord{
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

func syncSMSInbox(client *imapclient.Client, store *dbStore, days int) (int, int, error) {
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
		existingID, existingAttachmentText, found, err := store.findSMSByMessageID(messageID)
		if err == nil && found && strings.TrimSpace(existingAttachmentText) != "" {
			skipped++
			seenSet.AddNum(msg.Uid)
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
		if found {
			if err := store.updateSMSMissingData(existingID, smsRecord{
				SenderPhone:    senderPhone,
				MessageText:    inlineText,
				AttachmentText: attachmentText,
				AttachmentName: attachmentName,
				AttachmentData: attachmentData,
			}); err != nil {
				continue
			}
			stored++
			seenSet.AddNum(msg.Uid)
			continue
		}
		if err := store.insertSMS(smsRecord{
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
