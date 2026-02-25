package core

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
