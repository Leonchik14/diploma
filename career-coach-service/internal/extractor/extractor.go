package extractor

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/gen2brain/go-fitz"
)

type Extractor struct {
	maxChars int
}

func NewExtractor(maxChars int) *Extractor {
	return &Extractor{maxChars: maxChars}
}

func (e *Extractor) ExtractText(ctx context.Context, reader io.Reader, mimeType string) (string, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	if len(content) == 0 {
		return "", fmt.Errorf("file is empty")
	}

	var text string
	if strings.Contains(mimeType, "text/plain") || strings.HasSuffix(mimeType, "/txt") {
		text = string(content)
	} else if strings.Contains(mimeType, "pdf") {
		text, err = e.extractPDF(content)
		if err != nil {
			return "", err
		}
	} else if strings.Contains(mimeType, "wordprocessingml") || strings.Contains(mimeType, "docx") {
		text, err = e.extractDOCX(content)
		if err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("unsupported file type: %s", mimeType)
	}

	if len(text) > e.maxChars {
		text = text[:e.maxChars]
	}

	return text, nil
}

func (e *Extractor) extractPDF(content []byte) (string, error) {
	doc, err := fitz.NewFromMemory(content)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	var textBuilder strings.Builder
	numPages := doc.NumPage()
	
	for i := 0; i < numPages; i++ {
		pageText, err := doc.Text(i)
		if err != nil {
			return "", fmt.Errorf("failed to extract text from page %d: %w", i, err)
		}
		textBuilder.WriteString(pageText)
		textBuilder.WriteString("\n")
	}

	text := textBuilder.String()
	if len(text) == 0 {
		return "", fmt.Errorf("no text found in PDF")
	}

	return text, nil
}

func (e *Extractor) extractDOCX(content []byte) (string, error) {
	return "", fmt.Errorf("DOCX extraction not implemented - please use a DOCX library like github.com/lukasjarosch/go-docx")
}
