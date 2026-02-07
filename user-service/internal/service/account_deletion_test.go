package service

import (
	"context"
	"testing"

	"user-service/internal/config"
)

func TestAccountDeletionService_DeleteAccount(t *testing.T) {
	cfg := &config.Config{
		InternalAPIKey:           "test-key",
		MaterialsServiceAddr:     "localhost:50052",
		CoachServiceAddr:         "localhost:50053",
		JobsServiceAddr:          "localhost:50054",
		CalendarServiceAddr:      "localhost:50055",
	}

	svc := NewAccountDeletionService(cfg, nil)

	ctx := context.Background()

	err := svc.DeleteAccount(ctx, 999, "wrong-password")
	if err == nil {
		t.Error("Expected error for wrong password")
	}
}
