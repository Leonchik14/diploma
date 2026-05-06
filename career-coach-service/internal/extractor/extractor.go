package extractor

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"

	"github.com/gen2brain/go-fitz"
)

type Extractor struct {
	maxChars int
}

var (
	docxTextNodeRegex = regexp.MustCompile(`(?s)<w:t[^>]*>(.*?)</w:t>`)
	rtfControlRegex   = regexp.MustCompile(`\\[a-zA-Z]+\d* ?`)
	rtfBraceRegex     = regexp.MustCompile(`[{}]`)
	rtfHexRegex       = regexp.MustCompile(`\\'[0-9a-fA-F]{2}`)
)

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
	} else if strings.Contains(mimeType, "rtf") {
		text, err = e.extractRTF(content)
		if err != nil {
			return "", err
		}
	} else if strings.Contains(mimeType, "msword") || strings.Contains(mimeType, "application/doc") {
		return "", fmt.Errorf("legacy .doc format is not supported, please use .pdf, .docx, .rtf or .txt")
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
	readerAt := bytes.NewReader(content)
	zr, err := zip.NewReader(readerAt, int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("failed to open DOCX: %w", err)
	}

	var documentXML []byte
	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("failed to read DOCX xml: %w", err)
		}
		documentXML, err = io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return "", fmt.Errorf("failed to load DOCX xml: %w", err)
		}
		break
	}
	if len(documentXML) == 0 {
		return "", fmt.Errorf("word/document.xml not found in DOCX")
	}

	matches := docxTextNodeRegex.FindAllSubmatch(documentXML, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no text found in DOCX")
	}

	var b strings.Builder
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		b.WriteString(html.UnescapeString(string(m[1])))
		b.WriteString(" ")
	}
	out := strings.TrimSpace(strings.Join(strings.Fields(b.String()), " "))
	if out == "" {
		return "", fmt.Errorf("no text found in DOCX")
	}
	return out, nil
}

func (e *Extractor) extractRTF(content []byte) (string, error) {
	text := string(content)
	text = rtfHexRegex.ReplaceAllString(text, " ")
	text = rtfControlRegex.ReplaceAllString(text, " ")
	text = rtfBraceRegex.ReplaceAllString(text, " ")
	text = strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	if text == "" {
		return "", fmt.Errorf("no text found in RTF")
	}
	return text, nil
}
