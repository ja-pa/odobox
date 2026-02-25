package core

type OCRService interface {
	ExtractPDFText(pdfData []byte, lang string) (string, error)
}
