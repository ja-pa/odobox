package core

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
