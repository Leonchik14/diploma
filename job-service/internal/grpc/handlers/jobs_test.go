package handlers

import (
	"context"
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
	searchJobsFunc     func(ctx context.Context, userID string, page, perPage int) (*client.HHResponse, error)
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

func (m *mockSearchJobsService) GetVacancy(ctx context.Context, vacancyID string) (*client.HHVacancy, error) {
	return nil, nil
}

func (m *mockSearchJobsService) ListAreas(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockSearchJobsService) AddFavorite(ctx context.Context, userID, vacancyID string) error {
	return nil
}
func (m *mockSearchJobsService) RemoveFavorite(ctx context.Context, userID, vacancyID string) error {
	return nil
}
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
	svc := &mockSearchJobsService{
		searchJobsFunc: func(ctx context.Context, userID string, page, perPage int) (*client.HHResponse, error) {
			return &client.HHResponse{Items: nil, Found: 0, Pages: 0, Page: page}, nil
		},
	}
	h := NewJobsHandler(svc, logger)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "1", "x-internal-api-key", "key"))

	t.Run("negative page returns invalid argument", func(t *testing.T) {
		req := &pbjobs.SearchJobsRequest{Page: -1, PerPage: 10}
		_, err := h.SearchJobs(ctx, req)
		if err == nil {
			t.Fatal("expected error for negative page")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("per_page 0 returns invalid argument", func(t *testing.T) {
		req := &pbjobs.SearchJobsRequest{Page: 0, PerPage: 0}
		_, err := h.SearchJobs(ctx, req)
		if err == nil {
			t.Fatal("expected error for per_page=0")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got %v", err)
		}
	})
}
