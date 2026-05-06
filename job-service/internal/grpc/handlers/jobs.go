package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"job-service/internal/auth"
	"job-service/internal/client"
	"job-service/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbjobs "proto/jobs"
)

type JobsServiceInterface interface {
	SearchJobs(ctx context.Context, userID string, page, perPage int) (*client.HHResponse, error)
	GetVacancy(ctx context.Context, vacancyID string) (*client.HHVacancy, error)
	ListAreas(ctx context.Context) ([]string, error)
	GetFavoriteIDs(ctx context.Context, userID string) ([]string, error)
	AddFavorite(ctx context.Context, userID, vacancyID string) error
	RemoveFavorite(ctx context.Context, userID, vacancyID string) error
	ListFavorites(ctx context.Context, userID string) ([]*client.HHVacancy, error)
	DeleteUserData(ctx context.Context, userID string) error
}

type JobsHandler struct {
	pbjobs.UnimplementedJobsServiceServer
	service JobsServiceInterface
	logger  *slog.Logger
}

var vacancyIDRegex = regexp.MustCompile(`^\d+$`)

func NewJobsHandler(svc JobsServiceInterface, logger *slog.Logger) *JobsHandler {
	return &JobsHandler{
		service: svc,
		logger:  logger,
	}
}

func (h *JobsHandler) SearchJobs(ctx context.Context, req *pbjobs.SearchJobsRequest) (*pbjobs.SearchJobsResponse, error) {
	page := req.Page
	if page < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "page must be >= 0")
	}
	perPage := req.PerPage
	if perPage < 1 || perPage > 100 {
		return nil, status.Errorf(codes.InvalidArgument, "per_page must be between 1 and 100")
	}

	userID, err := auth.UserIDFromMetadata(ctx)
	if err != nil {
		return nil, err
	}

	h.logger.Info("SearchJobs request", "user_id", userID, "page", page, "per_page", perPage)

	resp, err := h.service.SearchJobs(ctx, userID, int(page), int(perPage))
	if err != nil {
		if errors.Is(err, service.ErrResumeProfileUnavailable) {
			return nil, status.Errorf(codes.FailedPrecondition, "resume profile not available")
		}
		if errors.Is(err, service.ErrResumeProfileIncomplete) {
			return nil, status.Errorf(codes.FailedPrecondition, "resume profile incomplete")
		}
		h.logger.Error("failed to search jobs", "error", err, "user_id", userID)
		return nil, status.Errorf(codes.Internal, "failed to search jobs: %v", err)
	}

	if len(resp.Items) == 0 {
		h.logger.Info("HH returned 0 vacancies", "user_id", userID, "found", resp.Found, "page", page, "per_page", perPage)
	}

	favoriteIDs, _ := h.service.GetFavoriteIDs(ctx, userID)
	out := convertHHResponse(resp, favoriteIDs, page, perPage)
	h.logger.Info("SearchJobs response", "user_id", userID, "found", out.Found, "pages", out.Pages, "page", out.Page, "per_page", out.PerPage, "items_count", len(out.Items))
	return out, nil
}

func convertHHResponse(resp *client.HHResponse, favoriteIDs []string, page, perPage int32) *pbjobs.SearchJobsResponse {
	favSet := make(map[string]bool)
	for _, id := range favoriteIDs {
		favSet[id] = true
	}

	items := make([]*pbjobs.Vacancy, 0, len(resp.Items))
	for _, item := range resp.Items {
		pbVacancy := convertHHVacancyToProtoWithFavorite(&item, favSet[item.ID])
		items = append(items, pbVacancy)
	}

	return &pbjobs.SearchJobsResponse{
		Items:   items,
		Found:   int32(resp.Found),
		Page:    page,
		Pages:   int32(resp.Pages),
		PerPage: perPage,
	}
}

func (h *JobsHandler) GetVacancy(ctx context.Context, req *pbjobs.GetVacancyRequest) (*pbjobs.GetVacancyResponse, error) {
	if req.VacancyId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "vacancy_id is required")
	}
	if !vacancyIDRegex.MatchString(req.VacancyId) {
		return nil, status.Errorf(codes.InvalidArgument, "vacancy_id must be numeric")
	}

	vacancy, err := h.service.GetVacancy(ctx, req.VacancyId)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "vacancy not found: %s", req.VacancyId)
		}
		h.logger.Error("failed to get vacancy", "error", err, "vacancy_id", req.VacancyId)
		return nil, status.Errorf(codes.Internal, "failed to get vacancy: %v", err)
	}

	return &pbjobs.GetVacancyResponse{
		Vacancy: convertHHVacancyToProtoWithFavorite(vacancy, false),
	}, nil
}

func (h *JobsHandler) ListAreas(ctx context.Context, req *pbjobs.ListAreasRequest) (*pbjobs.ListAreasResponse, error) {
	areaNames, err := h.service.ListAreas(ctx)
	if err != nil {
		h.logger.Error("failed to list areas", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to list areas")
	}
	return &pbjobs.ListAreasResponse{AreaNames: areaNames}, nil
}

func (h *JobsHandler) AddFavorite(ctx context.Context, req *pbjobs.AddFavoriteRequest) (*pbjobs.AddFavoriteResponse, error) {
	userID, err := auth.UserIDFromMetadata(ctx)
	if err != nil {
		return nil, err
	}

	if req.VacancyId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "vacancy_id is required")
	}
	if !vacancyIDRegex.MatchString(req.VacancyId) {
		return nil, status.Errorf(codes.InvalidArgument, "vacancy_id must be numeric")
	}

	if err := h.service.AddFavorite(ctx, userID, req.VacancyId); err != nil {
		h.logger.Error("failed to add favorite", "error", err, "user_id", userID, "vacancy_id", req.VacancyId)
		return nil, status.Errorf(codes.Internal, "failed to add favorite: %v", err)
	}

	return &pbjobs.AddFavoriteResponse{Success: true}, nil
}

func (h *JobsHandler) RemoveFavorite(ctx context.Context, req *pbjobs.RemoveFavoriteRequest) (*pbjobs.RemoveFavoriteResponse, error) {
	userID, err := auth.UserIDFromMetadata(ctx)
	if err != nil {
		return nil, err
	}

	if req.VacancyId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "vacancy_id is required")
	}
	if !vacancyIDRegex.MatchString(req.VacancyId) {
		return nil, status.Errorf(codes.InvalidArgument, "vacancy_id must be numeric")
	}

	if err := h.service.RemoveFavorite(ctx, userID, req.VacancyId); err != nil {
		h.logger.Error("failed to remove favorite", "error", err, "user_id", userID, "vacancy_id", req.VacancyId)
		return nil, status.Errorf(codes.Internal, "failed to remove favorite: %v", err)
	}

	return &pbjobs.RemoveFavoriteResponse{Success: true}, nil
}

func (h *JobsHandler) ListFavorites(ctx context.Context, req *pbjobs.ListFavoritesRequest) (*pbjobs.ListFavoritesResponse, error) {
	userID, err := auth.UserIDFromMetadata(ctx)
	if err != nil {
		return nil, err
	}

	vacancies, err := h.service.ListFavorites(ctx, userID)
	if err != nil {
		h.logger.Error("failed to list favorites", "error", err, "user_id", userID)
		return nil, status.Errorf(codes.Internal, "failed to list favorites: %v", err)
	}

	pbVacancies := make([]*pbjobs.Vacancy, len(vacancies))
	for i, v := range vacancies {
		pbVacancies[i] = convertHHVacancyToProtoWithFavorite(v, true)
	}

	return &pbjobs.ListFavoritesResponse{Vacancies: pbVacancies}, nil
}

func (h *JobsHandler) DeleteUserData(ctx context.Context, req *pbjobs.DeleteUserDataRequest) (*pbjobs.DeleteUserDataResponse, error) {
	userID := req.UserId
	if userID == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	if err := h.service.DeleteUserData(ctx, fmt.Sprintf("%d", userID)); err != nil {
		h.logger.Error("failed to delete user data", "user_id", userID, "error", err)
		return &pbjobs.DeleteUserDataResponse{Ok: true}, nil
	}

	return &pbjobs.DeleteUserDataResponse{Ok: true}, nil
}

func convertHHVacancyToProtoWithFavorite(v *client.HHVacancy, isFavorite bool) *pbjobs.Vacancy {
	pbVacancy := &pbjobs.Vacancy{
		Id:            v.ID,
		Name:          v.Name,
		Description:   v.GetDescription(),
		AlternateUrl:  v.URL,
		IsFavorite:    isFavorite,
		Archived:      v.Archived,
	}

	if v.Experience.Name != "" {
		pbVacancy.Experience = &v.Experience.Name
	}

	if v.Salary != nil {
		var from, to *int32
		if v.Salary.From != nil {
			val := int32(*v.Salary.From)
			from = &val
		}
		if v.Salary.To != nil {
			val := int32(*v.Salary.To)
			to = &val
		}
		pbVacancy.Salary = &pbjobs.Salary{
			From:     from,
			To:       to,
			Currency: v.Salary.Currency,
		}
	}

	emp := &pbjobs.Employer{Name: v.Employer.Name}
	if v.Employer.LogoURLs != nil {
		if v.Employer.LogoURLs.Size240 != "" {
			emp.LogoUrl = &v.Employer.LogoURLs.Size240
		} else if v.Employer.LogoURLs.Size90 != "" {
			emp.LogoUrl = &v.Employer.LogoURLs.Size90
		} else if v.Employer.LogoURLs.Original != "" {
			emp.LogoUrl = &v.Employer.LogoURLs.Original
		}
	}
	pbVacancy.Employer = emp

	pbVacancy.Area = &pbjobs.Area{
		Name: v.Area.Name,
	}

	return pbVacancy
}
