package core

import (
	"fmt"

	imapclient "github.com/emersion/go-imap/client"
)

func openIMAPClient(cfg imapConfig) (*imapclient.Client, error) {
	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	var client *imapclient.Client
	var err error
	if cfg.SSL {
		client, err = imapclient.DialTLS(address, nil)
	} else {
		client, err = imapclient.Dial(address)
	}
	if err != nil {
		return nil, fmt.Errorf("imap dial failed: %w", err)
	}
	if err := client.Login(cfg.Username, cfg.Password); err != nil {
		_ = client.Logout()
		return nil, fmt.Errorf("imap login failed: %w", err)
	}
	if _, err := client.Select(cfg.Folder, false); err != nil {
		_ = client.Logout()
		return nil, fmt.Errorf("imap select failed: %w", err)
	}
	return client, nil
}
