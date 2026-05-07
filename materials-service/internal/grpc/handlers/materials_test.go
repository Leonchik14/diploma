package handlers

import (
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNormalizeNodeName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantCode  codes.Code
		shouldErr bool
	}{
		{name: "trim and keep valid", input: "  Resume.pdf  ", want: "Resume.pdf"},
		{name: "empty after trim", input: "   ", wantCode: codes.InvalidArgument, shouldErr: true},
		{name: "forbidden chars", input: "bad/name", wantCode: codes.InvalidArgument, shouldErr: true},
		{name: "too long", input: strings.Repeat("a", maxNodeNameLen+1), wantCode: codes.InvalidArgument, shouldErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeNodeName(tt.input)
			if tt.shouldErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				st, ok := status.FromError(err)
				if !ok || st.Code() != tt.wantCode {
					t.Fatalf("expected gRPC code %v, got %v", tt.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestInferAndValidateUploadMIME(t *testing.T) {
	pdfHeader := []byte("%PDF-1.7\n")
	pngHeader := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	txtBody := []byte("hello world")

	tests := []struct {
		name      string
		filename  string
		content   []byte
		wantMIME  string
		wantCode  codes.Code
		shouldErr bool
	}{
		{name: "valid pdf", filename: "cv.pdf", content: pdfHeader, wantMIME: "application/pdf"},
		{name: "valid png", filename: "avatar.png", content: pngHeader, wantMIME: "image/png"},
		{name: "unsupported extension", filename: "x.exe", content: txtBody, wantCode: codes.InvalidArgument, shouldErr: true},
		{name: "mismatch content and extension", filename: "cv.pdf", content: txtBody, wantCode: codes.InvalidArgument, shouldErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := inferAndValidateUploadMIME(tt.filename, tt.content)
			if tt.shouldErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				st, ok := status.FromError(err)
				if !ok || st.Code() != tt.wantCode {
					t.Fatalf("expected gRPC code %v, got %v", tt.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantMIME {
				t.Fatalf("expected mime %q, got %q", tt.wantMIME, got)
			}
		})
	}
}
