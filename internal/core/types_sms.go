package core

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

type ListSMSHistoryRequest struct {
	Days int `json:"days"`
}

type SMSHistoryItem struct {
	ID               int    `json:"id"`
	Direction        string `json:"direction"`
	OccurredAt       string `json:"occurred_at"`
	Counterparty     string `json:"counterparty"`
	MessageText      string `json:"message_text"`
	Subject          string `json:"subject"`
	SenderID         string `json:"sender_id"`
	Success          bool   `json:"success"`
	ProviderResponse string `json:"provider_response"`
	ErrorMessage     string `json:"error_message"`
}

type ListSMSHistoryResponse struct {
	Items []SMSHistoryItem `json:"items"`
	Count int              `json:"count"`
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
