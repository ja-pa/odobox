package core

import "time"

type MailSearchCriteria struct {
	Since time.Time
	From  string
}

type MailEnvelope struct {
	Subject string
	Date    time.Time
	From    []string
}

type FetchedMessage struct {
	UID      uint32
	Seq      uint32
	Envelope MailEnvelope
	Raw      []byte
}

type MailGateway interface {
	Search(criteria MailSearchCriteria) ([]uint32, error)
	FetchEnvelopes(ids []uint32) ([]FetchedMessage, error)
	FetchBodies(ids []uint32, useUID bool) ([]FetchedMessage, error)
	MarkSeen(uids []uint32) error
	Close() error
}

type MailGatewayFactory interface {
	Open(host string, port int, ssl bool, username, password, folder string) (MailGateway, error)
}

type SyncStore interface {
	Close() error
	VoicemailExists(messageID string) (bool, error)
	InsertVoicemail(rec SyncVoicemailRecord) error
	FindSMSByMessageID(messageID string) (id int, attachmentText string, found bool, err error)
	UpdateSMSMissingData(id int, rec SyncSMSRecord) error
	InsertSMS(rec SyncSMSRecord) error
}

type SyncStoreFactory interface {
	Open(path string) (SyncStore, error)
}

type OCRService interface {
	ExtractPDFText(pdfData []byte, lang string) (string, error)
}

type BackendDeps struct {
	MailGatewayFactory MailGatewayFactory
	SyncStoreFactory   SyncStoreFactory
	OCRService         OCRService
}

type SyncVoicemailRecord struct {
	MessageID      string
	Date           *time.Time
	Subject        string
	MessageText    string
	AttachmentName string
	AttachmentData []byte
	AudioDuration  *int
}

type SyncSMSRecord struct {
	MessageID      string
	Date           *time.Time
	Subject        string
	SenderPhone    string
	MessageText    string
	AttachmentText string
	AttachmentName string
	AttachmentData []byte
}
