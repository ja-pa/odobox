package core

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const DefaultHTTPAPIPort = 51731

type HTTPAPIConfig struct {
	Enabled bool
	Port    int
	Token   string
}

func (b *Backend) ListVoicemails(req ListVoicemailsRequest) (ListVoicemailsResponse, error) {
	return b.listVoicemails(req)
}

func (b *Backend) Sync(days int) (SyncResponse, error) {
	return b.sync(days)
}

func (b *Backend) SetVoicemailChecked(id int, checked bool) (UpdateCheckedResponse, error) {
	return b.updateChecked(id, checked)
}

func (b *Backend) ListSMSMessages(req ListSMSMessagesRequest) (ListSMSMessagesResponse, error) {
	return b.listSMSMessages(req)
}

func (b *Backend) SetSMSMessageChecked(id int, checked bool) (UpdateCheckedResponse, error) {
	return b.updateSMSChecked(id, checked)
}

func (b *Backend) GetVoicemailAudioDataURL(id int) (string, error) {
	return b.getVoicemailAudioDataURL(id)
}

func (b *Backend) GetSettings() (SettingsResponse, error) {
	return b.getSettings()
}

func (b *Backend) PatchSettings(req PatchSettingsRequest) (PatchSettingsResponse, error) {
	return b.patchSettings(req)
}

func (b *Backend) SendSMS(req SendSMSRequest) (SendSMSResponse, error) {
	return b.sendSMS(req)
}

func (b *Backend) GetOdorikBalance() (OdorikBalanceResponse, error) {
	return b.getOdorikBalance()
}

func (b *Backend) ListSMSTemplates() ([]SMSTemplate, error) {
	return b.listSMSTemplates()
}

func (b *Backend) CreateSMSTemplate(req CreateSMSTemplateRequest) (SMSTemplate, error) {
	return b.createSMSTemplate(req)
}

func (b *Backend) UpdateSMSTemplate(req UpdateSMSTemplateRequest) (SMSTemplate, error) {
	return b.updateSMSTemplate(req)
}

func (b *Backend) DeleteSMSTemplate(id int) (DeleteSMSTemplateResponse, error) {
	return b.deleteSMSTemplate(id)
}

func (b *Backend) ListContacts() ([]ContactInfo, error) {
	return b.listContacts()
}

func (b *Backend) ImportVCF(req ImportVCFRequest) (ImportVCFResponse, error) {
	return b.importVCF(req)
}

func (b *Backend) ExportVCF() (ExportVCFResponse, error) {
	return b.exportVCF()
}

func (b *Backend) CreateContact(req CreateContactRequest) (ContactInfo, error) {
	return b.createContact(req)
}

func (b *Backend) UpdateContact(req UpdateContactRequest) (ContactInfo, error) {
	return b.updateContact(req)
}

func (b *Backend) DeleteContact(id int) (DeleteContactResponse, error) {
	return b.deleteContact(id)
}

func (b *Backend) DebugIMAP(days int, limit int) ([]IMAPDebugItem, error) {
	return b.debugIMAP(days, limit)
}

func (b *Backend) DebugIMAPMessage(uid uint32) (IMAPMessageDebug, error) {
	return b.debugIMAPMessage(uid)
}

func (b *Backend) ResolveConfigPath() string {
	return b.resolveConfigPath()
}

func (b *Backend) ResolveDBPathFromCurrentConfig() (string, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return "", err
	}
	return b.resolveDBPath(cfg), nil
}

func (b *Backend) LoadHTTPAPIConfig() (HTTPAPIConfig, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return HTTPAPIConfig{}, err
	}
	flaskEnabled := parseBoolLoose(cfg.get("flask", "debug", "false"), false)
	enabled := parseBoolLoose(cfg.get("http_api", "enabled", strconv.FormatBool(flaskEnabled)), flaskEnabled)
	if raw := strings.TrimSpace(os.Getenv("ENABLE_HTTP_API")); raw != "" {
		enabled = parseBoolLoose(raw, enabled)
	}

	portRaw := strings.TrimSpace(cfg.get("http_api", "port", cfg.get("flask", "port", strconv.Itoa(DefaultHTTPAPIPort))))
	if raw := strings.TrimSpace(os.Getenv("HTTP_API_PORT")); raw != "" {
		portRaw = raw
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil || port <= 0 || port > 65535 {
		return HTTPAPIConfig{}, fmt.Errorf("http_api.port must be a valid TCP port")
	}

	token := strings.TrimSpace(cfg.get("http_api", "token", ""))
	if envToken := strings.TrimSpace(os.Getenv("HTTP_API_TOKEN")); envToken != "" {
		token = envToken
	}
	return HTTPAPIConfig{
		Enabled: enabled,
		Port:    port,
		Token:   token,
	}, nil
}
