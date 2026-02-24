package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	ocradapter "OdorikCentral/internal/adapters/ocr"
)

func main() {
	input := flag.String("input", "", "Input file path (image or PDF)")
	lang := flag.String("lang", "ces+eng", "Tesseract language(s), e.g. ces, eng, ces+eng")
	output := flag.String("output", "", "Optional output text file (default: stdout)")
	flag.Parse()

	service := ocradapter.NewService()
	text, err := service.ExtractFileText(*input, *lang)
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
