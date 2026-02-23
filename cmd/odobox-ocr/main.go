package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	tesseractPath = "/usr/bin/tesseract"
	pdftoppmPath  = "/usr/bin/pdftoppm"
)

func main() {
	input := flag.String("input", "", "Input file path (image or PDF)")
	lang := flag.String("lang", "ces+eng", "Tesseract language(s), e.g. ces, eng, ces+eng")
	output := flag.String("output", "", "Optional output text file (default: stdout)")
	flag.Parse()

	text, err := runOCR(*input, *lang)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ocr failed: %v\n", err)
		os.Exit(1)
	}

	if strings.TrimSpace(*output) == "" {
		fmt.Print(text)
		return
	}

	if err := os.WriteFile(*output, []byte(text), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OCR text written to %s\n", *output)
}

func runOCR(inputPath string, lang string) (string, error) {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		return "", errors.New("input is required")
	}
	if _, err := os.Stat(inputPath); err != nil {
		return "", fmt.Errorf("cannot access input: %w", err)
	}

	if _, err := os.Stat(tesseractPath); err != nil {
		return "", fmt.Errorf("tesseract not found at %s: %w", tesseractPath, err)
	}

	ext := strings.ToLower(filepath.Ext(inputPath))
	if ext == ".pdf" {
		return runOCRPDF(inputPath, lang)
	}
	return runOCRImage(inputPath, lang)
}

func runOCRImage(inputPath string, lang string) (string, error) {
	cmd := exec.Command(tesseractPath, inputPath, "stdout", "-l", lang, "--psm", "6")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tesseract image OCR error: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func runOCRPDF(pdfPath string, lang string) (string, error) {
	if _, err := os.Stat(pdftoppmPath); err != nil {
		return "", fmt.Errorf("pdftoppm not found at %s: %w", pdftoppmPath, err)
	}

	tmpDir, err := os.MkdirTemp("", "odobox-ocr-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	prefix := filepath.Join(tmpDir, "page")
	convert := exec.Command(pdftoppmPath, "-r", "300", "-png", pdfPath, prefix)
	convertOut, err := convert.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pdf to image conversion failed: %w (%s)", err, strings.TrimSpace(string(convertOut)))
	}

	pages, err := listPNGPages(tmpDir)
	if err != nil {
		return "", err
	}
	if len(pages) == 0 {
		return "", errors.New("no pages produced from PDF conversion")
	}

	var out strings.Builder
	for i, page := range pages {
		text, ocrErr := runOCRImage(page, lang)
		if ocrErr != nil {
			return "", fmt.Errorf("page %d OCR failed: %w", i+1, ocrErr)
		}
		if i > 0 {
			out.WriteString("\n\n----- PAGE ")
			out.WriteString(fmt.Sprintf("%d", i+1))
			out.WriteString(" -----\n\n")
		}
		out.WriteString(text)
	}
	return out.String(), nil
}

func listPNGPages(dir string) ([]string, error) {
	pages := []string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".png") {
			pages = append(pages, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(pages)
	return pages, nil
}
