package core

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

type UpdateCheckedResponse struct {
	Status    string `json:"status"`
	ID        int    `json:"id"`
	IsChecked bool   `json:"is_checked"`
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
