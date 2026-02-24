package imap

import (
	"fmt"
	"io"
	"strings"

	"OdorikCentral/internal/core"
	imap "github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
)

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

type gateway struct {
	client *imapclient.Client
}

func (f *Factory) Open(host string, port int, ssl bool, username, password, folder string) (core.MailGateway, error) {
	address := fmt.Sprintf("%s:%d", host, port)
	var client *imapclient.Client
	var err error
	if ssl {
		client, err = imapclient.DialTLS(address, nil)
	} else {
		client, err = imapclient.Dial(address)
	}
	if err != nil {
		return nil, fmt.Errorf("imap dial failed: %w", err)
	}
	if err := client.Login(username, password); err != nil {
		_ = client.Logout()
		return nil, fmt.Errorf("imap login failed: %w", err)
	}
	if _, err := client.Select(folder, false); err != nil {
		_ = client.Logout()
		return nil, fmt.Errorf("imap select failed: %w", err)
	}
	return &gateway{client: client}, nil
}

func (g *gateway) Close() error {
	return g.client.Logout()
}

func (g *gateway) Search(criteria core.MailSearchCriteria) ([]uint32, error) {
	c := imap.NewSearchCriteria()
	c.Since = criteria.Since
	if strings.TrimSpace(criteria.From) != "" {
		c.Header = map[string][]string{"From": {criteria.From}}
	}
	return g.client.Search(c)
}

func (g *gateway) FetchEnvelopes(ids []uint32) ([]core.FetchedMessage, error) {
	if len(ids) == 0 {
		return []core.FetchedMessage{}, nil
	}
	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)
	items := []imap.FetchItem{imap.FetchUid, imap.FetchEnvelope}
	messages := make(chan *imap.Message, 64)
	fetchErr := make(chan error, 1)
	go func() { fetchErr <- g.client.Fetch(seqset, items, messages) }()

	out := make([]core.FetchedMessage, 0, len(ids))
	for msg := range messages {
		if msg == nil {
			continue
		}
		out = append(out, core.FetchedMessage{
			UID:      msg.Uid,
			Seq:      msg.SeqNum,
			Envelope: envelopeFromIMAP(msg.Envelope),
		})
	}
	if err := <-fetchErr; err != nil {
		return nil, err
	}
	return out, nil
}

func (g *gateway) FetchBodies(ids []uint32, useUID bool) ([]core.FetchedMessage, error) {
	if len(ids) == 0 {
		return []core.FetchedMessage{}, nil
	}
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchUid, imap.FetchEnvelope, section.FetchItem()}
	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)
	messages := make(chan *imap.Message, 64)
	fetchErr := make(chan error, 1)
	go func() {
		if useUID {
			fetchErr <- g.client.UidFetch(seqset, items, messages)
			return
		}
		fetchErr <- g.client.Fetch(seqset, items, messages)
	}()

	out := make([]core.FetchedMessage, 0, len(ids))
	for msg := range messages {
		if msg == nil {
			continue
		}
		body := msg.GetBody(section)
		if body == nil {
			continue
		}
		raw, err := io.ReadAll(body)
		if err != nil {
			continue
		}
		out = append(out, core.FetchedMessage{
			UID:      msg.Uid,
			Seq:      msg.SeqNum,
			Envelope: envelopeFromIMAP(msg.Envelope),
			Raw:      raw,
		})
	}
	if err := <-fetchErr; err != nil {
		return nil, err
	}
	return out, nil
}

func (g *gateway) MarkSeen(uids []uint32) error {
	if len(uids) == 0 {
		return nil
	}
	set := new(imap.SeqSet)
	set.AddNum(uids...)
	return g.client.UidStore(set, imap.FormatFlagsOp(imap.AddFlags, true), []interface{}{imap.SeenFlag}, nil)
}

func envelopeFromIMAP(env *imap.Envelope) core.MailEnvelope {
	if env == nil {
		return core.MailEnvelope{}
	}
	out := core.MailEnvelope{
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
