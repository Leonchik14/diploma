package handlers

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"job-service/internal/client"
	"job-service/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pbjobs "proto/jobs"
)

type mockSearchJobsService struct {
	searchJobsFunc    func(ctx context.Context, userID string, page, perPage int) (*client.HHResponse, error)
	getFavoriteIDsFunc func(ctx context.Context, userID string) ([]string, error)
}

func (m *mockSearchJobsService) SearchJobs(ctx context.Context, userID string, page, perPage int) (*client.HHResponse, error) {
	if m.searchJobsFunc != nil {
		return m.searchJobsFunc(ctx, userID, page, perPage)
	}
	return nil, nil
}

func (m *mockSearchJobsService) GetFavoriteIDs(ctx context.Context, userID string) ([]string, error) {
	if m.getFavoriteIDsFunc != nil {
		return m.getFavoriteIDsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockSearchJobsService) AddFavorite(ctx context.Context, userID, vacancyID string) error { return nil }
func (m *mockSearchJobsService) RemoveFavorite(ctx context.Context, userID, vacancyID string) error { return nil }
func (m *mockSearchJobsService) ListFavorites(ctx context.Context, userID string) ([]*client.HHVacancy, error) {
	return nil, nil
}
func (m *mockSearchJobsService) DeleteUserData(ctx context.Context, userID string) error { return nil }

func TestSearchJobs_RejectsWhenNoUserIDMetadata(t *testing.T) {
	logger := slog.Default()
	h := NewJobsHandler(&mockSearchJobsService{}, logger)
	ctx := context.Background()
	req := &pbjobs.SearchJobsRequest{Page: 0, PerPage: 10}

	_, err := h.SearchJobs(ctx, req)
	if err == nil {
		t.Fatal("expected error when no x-user-id metadata")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %v", err)
	}
}

func TestSearchJobs_RejectsWhenProfileMissing(t *testing.T) {
	logger := slog.Default()
	svc := &mockSearchJobsService{
		searchJobsFunc: func(ctx context.Context, userID string, page, perPage int) (*client.HHResponse, error) {
			return nil, service.ErrResumeProfileUnavailable
		},
	}
	h := NewJobsHandler(svc, logger)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "1", "x-internal-api-key", "key"))
	req := &pbjobs.SearchJobsRequest{Page: 0, PerPage: 10}

	_, err := h.SearchJobs(ctx, req)
	if err == nil {
		t.Fatal("expected error when profile missing")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status, got %v", err)
	}
	if st.Code() != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition, got %v", st.Code())
	}
}

func TestSearchJobs_RejectsWhenTargetRolesEmpty(t *testing.T) {
	logger := slog.Default()
	svc := &mockSearchJobsService{
		searchJobsFunc: func(ctx context.Context, userID string, page, perPage int) (*client.HHResponse, error) {
			return nil, service.ErrResumeProfileIncomplete
		},
	}
	h := NewJobsHandler(svc, logger)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "1", "x-internal-api-key", "key"))
	req := &pbjobs.SearchJobsRequest{Page: 0, PerPage: 10}

	_, err := h.SearchJobs(ctx, req)
	if err == nil {
		t.Fatal("expected error when target_roles empty")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition, got %v", err)
	}
}

func TestSearchJobs_PaginationBounds(t *testing.T) {
	logger := slog.Default()
	var gotPage, gotPerPage int
	svc := &mockSearchJobsService{
		searchJobsFunc: func(ctx context.Context, userID string, page, perPage int) (*client.HHResponse, error) {
			gotPage, gotPerPage = page, perPage
			return &client.HHResponse{Items: nil, Found: 0, Pages: 0, Page: page}, nil
		},
	}
	h := NewJobsHandler(svc, logger)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "1", "x-internal-api-key", "key"))

	t.Run("negative page clamped to 0", func(t *testing.T) {
		req := &pbjobs.SearchJobsRequest{Page: -1, PerPage: 10}
		_, err := h.SearchJobs(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		if gotPage != 0 {
			t.Errorf("expected page 0, got %d", gotPage)
		}
		if gotPerPage != 10 {
			t.Errorf("expected perPage 10, got %d", gotPerPage)
		}
	})

	t.Run("per_page 0 defaults to 10", func(t *testing.T) {
		req := &pbjobs.SearchJobsRequest{Page: 0, PerPage: 0}
		_, err := h.SearchJobs(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		if gotPerPage != 10 {
			t.Errorf("expected perPage 10, got %d", gotPerPage)
		}
	})

	t.Run("per_page over 100 clamped to 100", func(t *testing.T) {
		req := &pbjobs.SearchJobsRequest{Page: 0, PerPage: 200}
		_, err := h.SearchJobs(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		if gotPerPage != 100 {
			t.Errorf("expected perPage 100, got %d", gotPerPage)
		}
	})
}
