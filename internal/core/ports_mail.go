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
