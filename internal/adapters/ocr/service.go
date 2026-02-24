package ocr

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"OdorikCentral/internal/core"
)

const (
	defaultTesseractBinary = "/usr/bin/tesseract"
	defaultPdftoppmBinary  = "/usr/bin/pdftoppm"
)

type Service struct {
	tesseractBinary string
	pdftoppmBinary  string
}

func NewService() *Service {
	tesseract := strings.TrimSpace(os.Getenv("ODOBOX_TESSERACT_BIN"))
	if tesseract == "" {
		tesseract = defaultTesseractBinary
	}
	pdftoppm := strings.TrimSpace(os.Getenv("ODOBOX_PDFTOPPM_BIN"))
	if pdftoppm == "" {
		pdftoppm = defaultPdftoppmBinary
	}
	return &Service{
		tesseractBinary: tesseract,
		pdftoppmBinary:  pdftoppm,
	}
}

var _ core.OCRService = (*Service)(nil)

func (s *Service) ExtractFileText(inputPath string, lang string) (string, error) {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		return "", errors.New("input is required")
	}
	if _, err := os.Stat(inputPath); err != nil {
		return "", fmt.Errorf("cannot access input: %w", err)
	}
	if strings.TrimSpace(lang) == "" {
		lang = "ces+eng"
	}
	if _, err := os.Stat(s.tesseractBinary); err != nil {
		return "", fmt.Errorf("tesseract not found: %w", err)
	}

	if strings.EqualFold(filepath.Ext(inputPath), ".pdf") {
		pdfData, err := os.ReadFile(inputPath)
		if err != nil {
			return "", err
		}
		return s.ExtractPDFText(pdfData, lang)
	}

	return s.extractImageText(inputPath, lang)
}

func (s *Service) ExtractPDFText(pdfData []byte, lang string) (string, error) {
	if len(pdfData) == 0 {
		return "", fmt.Errorf("empty PDF data")
	}
	if strings.TrimSpace(lang) == "" {
		lang = "ces+eng"
	}
	if _, err := os.Stat(s.tesseractBinary); err != nil {
		return "", fmt.Errorf("tesseract not found: %w", err)
	}
	if _, err := os.Stat(s.pdftoppmBinary); err != nil {
		return "", fmt.Errorf("pdftoppm not found: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "odobox-ocr-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "input.pdf")
	if err := os.WriteFile(pdfPath, pdfData, 0o600); err != nil {
		return "", err
	}

	prefix := filepath.Join(tmpDir, "page")
	conv := exec.Command(s.pdftoppmBinary, "-r", "300", "-png", pdfPath, prefix)
	convOut, err := conv.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pdftoppm failed: %w (%s)", err, strings.TrimSpace(string(convOut)))
	}

	pages, err := filepath.Glob(filepath.Join(tmpDir, "page-*.png"))
	if err != nil {
		return "", err
	}
	sort.Strings(pages)
	if len(pages) == 0 {
		return "", fmt.Errorf("no PNG pages produced")
	}

	var out strings.Builder
	for i, page := range pages {
		txt, ocrErr := s.extractImageText(page, lang)
		if ocrErr != nil {
			return "", fmt.Errorf("tesseract failed on page %d: %w", i+1, ocrErr)
		}
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString(strings.TrimSpace(txt))
		out.WriteString("\n")
	}
	return strings.TrimSpace(out.String()), nil
}

func (s *Service) extractImageText(imagePath string, lang string) (string, error) {
	cmd := exec.Command(s.tesseractBinary, imagePath, "stdout", "-l", lang, "--psm", "6")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tesseract image OCR error: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
