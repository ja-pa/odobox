package core

import "time"

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
