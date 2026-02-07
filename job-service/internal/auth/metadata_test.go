package auth

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUserIDFromMetadata_Missing(t *testing.T) {
	ctx := context.Background()
	_, err := UserIDFromMetadata(ctx)
	if err == nil {
		t.Fatal("expected error when no metadata")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %v", err)
	}
}

func TestUserIDFromMetadata_EmptyValue(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(MetadataUserID, ""))
	_, err := UserIDFromMetadata(ctx)
	if err == nil {
		t.Fatal("expected error when x-user-id empty")
	}
}

func TestUserIDFromMetadata_Ok(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(MetadataUserID, "42"))
	got, err := UserIDFromMetadata(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "42" {
		t.Errorf("expected 42, got %q", got)
	}
}

func TestUserIDUintFromMetadata_InvalidFormat(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(MetadataUserID, "abc"))
	_, err := UserIDUintFromMetadata(ctx)
	if err == nil {
		t.Fatal("expected error for non-numeric user id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

func TestUserIDUintFromMetadata_Ok(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(MetadataUserID, "123"))
	got, err := UserIDUintFromMetadata(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != 123 {
		t.Errorf("expected 123, got %d", got)
	}
}
