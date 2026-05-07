package service

import (
	"strings"
	"testing"
)

func TestGenerateObjectKey(t *testing.T) {
	svc := &Service{}

	key := svc.generateObjectKey(12, 34)
	if !strings.HasPrefix(key, "user/12/34/") {
		t.Fatalf("expected key prefix user/12/34/, got %q", key)
	}

	parts := strings.Split(key, "/")
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts in key, got %d (%q)", len(parts), key)
	}
	if parts[3] == "" {
		t.Fatal("expected uuid suffix in key")
	}
}
