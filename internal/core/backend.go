package core

import "gopkg.in/ini.v1"

const (
	defaultDBPath      = "./voicemail.db"
	defaultCfgPath     = "./config.ini"
	defaultOCRLanguage = "ces+eng"
)

type appConfig struct {
	file *ini.File
}

type imapConfig struct {
	Host     string
	Port     int
	SSL      bool
	Username string
	Password string
	Folder   string
}

type cleanerConfig struct {
	KeepLineRegex      string
	RemoveRegexes      []string
	CollapseBlankLines bool
	VersionV1Regex     string
	VersionV2Regex     string
}

type parserConfig struct {
	CallerPhoneRegex string
}

type smsParserConfig struct {
	TextExtractRegex string
}

type smsConfig struct {
	User      string
	Password  string
	DefaultID string
}

type ListVoicemailsRequest struct {
	Days    int    `json:"days"`
	Clean   bool   `json:"clean"`
	Checked string `json:"checked"`
	Version string `json:"version"`
}

type VoicemailItem struct {
	ID            int          `json:"id"`
	DateReceived  string       `json:"date_received"`
	Subject       string       `json:"subject"`
	CallerPhone   *string      `json:"caller_phone"`
	MessageText   string       `json:"message_text"`
	IsChecked     bool         `json:"is_checked"`
	Attachment    string       `json:"attachment_name"`
	MP3Downloaded bool         `json:"mp3_downloaded"`
	AudioSeconds  *int         `json:"audio_duration_s"`
	Contact       *ContactInfo `json:"contact,omitempty"`
}

type ListVoicemailsResponse struct {
	Items   []VoicemailItem `json:"items"`
	Count   int             `json:"count"`
	Clean   bool            `json:"clean"`
	Version string          `json:"version"`
}

type ListSMSMessagesRequest struct {
	Days    int    `json:"days"`
	Checked string `json:"checked"`
}

type SMSMessageItem struct {
	ID             int          `json:"id"`
	DateReceived   string       `json:"date_received"`
	Subject        string       `json:"subject"`
	SenderPhone    *string      `json:"sender_phone"`
	MessageText    string       `json:"message_text"`
	AttachmentText string       `json:"attachment_text"`
	IsChecked      bool         `json:"is_checked"`
	Attachment     string       `json:"attachment_name"`
	PDFDownloaded  bool         `json:"pdf_downloaded"`
	Contact        *ContactInfo `json:"contact,omitempty"`
}

type ListSMSMessagesResponse struct {
	Items []SMSMessageItem `json:"items"`
	Count int              `json:"count"`
}

type SyncResponse struct {
	Status            string `json:"status"`
	Days              int    `json:"days"`
	Stored            int    `json:"stored"`
	SkippedDuplicates int    `json:"skipped_duplicates"`
	VoicemailStored   int    `json:"voicemail_stored"`
	SMSStored         int    `json:"sms_stored"`
	VoicemailSkipped  int    `json:"voicemail_skipped"`
	SMSSkipped        int    `json:"sms_skipped"`
}

type IMAPDebugItem struct {
	Seq         uint32 `json:"seq"`
	UID         uint32 `json:"uid"`
	Date        string `json:"date"`
	From        string `json:"from"`
	Subject     string `json:"subject"`
	SMSLike     bool   `json:"sms_like"`
	Voicemail   bool   `json:"voicemail_like"`
	HasFromHost bool   `json:"has_odorik_host"`
}

type IMAPPartDebug struct {
	Kind        string `json:"kind"`
	ContentType string `json:"content_type"`
	Filename    string `json:"filename"`
	SizeBytes   int    `json:"size_bytes"`
	Sample      string `json:"sample"`
}

type IMAPMessageDebug struct {
	Seq              uint32          `json:"seq"`
	UID              uint32          `json:"uid"`
	From             string          `json:"from"`
	Subject          string          `json:"subject"`
	Date             string          `json:"date"`
	Parts            []IMAPPartDebug `json:"parts"`
	SMSPDFCount      int             `json:"sms_pdf_count"`
	SMSInlineTextLen int             `json:"sms_inline_text_len"`
}

type UpdateCheckedResponse struct {
	Status    string `json:"status"`
	ID        int    `json:"id"`
	IsChecked bool   `json:"is_checked"`
}

type SettingsResponse struct {
	Settings         map[string]map[string]any `json:"settings"`
	EditableSections []string                  `json:"editable_sections"`
}

type PatchSettingsRequest struct {
	Settings map[string]map[string]any `json:"settings"`
}

type PatchSettingsResponse struct {
	Status   string                    `json:"status"`
	Settings map[string]map[string]any `json:"settings"`
}

type SendSMSRequest struct {
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
	Sender    string `json:"sender"`
}

type SendSMSResponse struct {
	Status           string `json:"status"`
	Recipient        string `json:"recipient"`
	Sender           string `json:"sender"`
	Encoding         string `json:"encoding"`
	CharsUsed        int    `json:"chars_used"`
	MaxSingleChars   int    `json:"max_single_chars"`
	ProviderResponse string `json:"provider_response"`
	SentAt           string `json:"sent_at"`
}

type OdorikBalanceResponse struct {
	Status           string `json:"status"`
	Balance          string `json:"balance"`
	Currency         string `json:"currency"`
	ProviderResponse string `json:"provider_response"`
	UpdatedAt        string `json:"updated_at"`
}

type ContactInfo struct {
	ID        int    `json:"id"`
	FullName  string `json:"full_name"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Org       string `json:"org"`
	Note      string `json:"note"`
	VCard     string `json:"vcard"`
	UpdatedAt string `json:"updated_at"`
}

type ImportVCFRequest struct {
	Content string `json:"content"`
}

type ImportVCFResponse struct {
	Status    string `json:"status"`
	Imported  int    `json:"imported"`
	Updated   int    `json:"updated"`
	Skipped   int    `json:"skipped"`
	Processed int    `json:"processed"`
}

type ExportVCFResponse struct {
	Status  string `json:"status"`
	Content string `json:"content"`
	Count   int    `json:"count"`
}

type CreateContactRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Org      string `json:"org"`
	Note     string `json:"note"`
}

type UpdateContactRequest struct {
	ID       int    `json:"id"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Org      string `json:"org"`
	Note     string `json:"note"`
}

type DeleteContactResponse struct {
	Status string `json:"status"`
	ID     int    `json:"id"`
}

type SMSTemplate struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CreateSMSTemplateRequest struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

type UpdateSMSTemplateRequest struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Body string `json:"body"`
}

type DeleteSMSTemplateResponse struct {
	Status string `json:"status"`
	ID     int    `json:"id"`
}

type Backend struct {
	configPath         string
	mailGatewayFactory MailGatewayFactory
	syncStoreFactory   SyncStoreFactory
	ocrService         OCRService
}

func NewBackend(configPath string) *Backend {
	return &Backend{configPath: configPath}
}

func NewBackendWithDeps(configPath string, deps BackendDeps) *Backend {
	return &Backend{
		configPath:         configPath,
		mailGatewayFactory: deps.MailGatewayFactory,
		syncStoreFactory:   deps.SyncStoreFactory,
		ocrService:         deps.OCRService,
	}
}
