package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func ocrPDFData(pdfData []byte, lang string) (string, error) {
	if len(pdfData) == 0 {
		return "", fmt.Errorf("empty PDF data")
	}
	if strings.TrimSpace(lang) == "" {
		lang = defaultOCRLanguage
	}
	if _, err := os.Stat(tesseractBinary); err != nil {
		return "", fmt.Errorf("tesseract not found: %w", err)
	}
	if _, err := os.Stat(pdftoppmBinary); err != nil {
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
	conv := exec.Command(pdftoppmBinary, "-r", "300", "-png", pdfPath, prefix)
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
		cmd := exec.Command(tesseractBinary, page, "stdout", "-l", lang, "--psm", "6")
		txt, ocrErr := cmd.CombinedOutput()
		if ocrErr != nil {
			return "", fmt.Errorf("tesseract failed on page %d: %w (%s)", i+1, ocrErr, strings.TrimSpace(string(txt)))
		}
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString(strings.TrimSpace(string(txt)))
		out.WriteString("\n")
	}
	return strings.TrimSpace(out.String()), nil
}
