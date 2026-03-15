package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"job-service/internal/client"
	"job-service/internal/repository"
)

var (
	ErrResumeProfileIncomplete  = errors.New("resume profile incomplete")
	ErrResumeProfileUnavailable = errors.New("resume profile not available")
)

type Service struct {
	userClient    *client.UserClient
	hhClient      *client.HHClient
	favoritesRepo *repository.FavoritesRepo
}

func NewService(userClient *client.UserClient, hhClient *client.HHClient, favoritesRepo *repository.FavoritesRepo) *Service {
	return &Service{
		userClient:    userClient,
		hhClient:      hhClient,
		favoritesRepo: favoritesRepo,
	}
}

func (s *Service) SearchJobs(ctx context.Context, userID string, page, perPage int) (*client.HHResponse, error) {
	result, err := s.userClient.GetResumeProfile(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrResumeProfileUnavailable, err)
	}
	if result == nil || result.Profile == nil {
		return nil, ErrResumeProfileUnavailable
	}
	if result.Status == "DRAFT" && len(result.Profile.TargetRoles) == 0 {
		return nil, ErrResumeProfileIncomplete
	}

	params := client.BuildHHQuery(result.Profile, page, perPage)
	hhResp, err := s.hhClient.SearchVacancies(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search vacancies: %w", err)
	}
	return hhResp, nil
}

func (s *Service) GetVacancy(ctx context.Context, vacancyID string) (*client.HHVacancy, error) {
	if vacancyID == "" {
		return nil, fmt.Errorf("vacancy_id is required")
	}
	return s.hhClient.GetVacancyByID(ctx, vacancyID)
}

func (s *Service) AddFavorite(ctx context.Context, userID string, vacancyID string) error {
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user_id: %w", err)
	}
	return s.favoritesRepo.Add(ctx, uid, vacancyID)
}

func (s *Service) RemoveFavorite(ctx context.Context, userID string, vacancyID string) error {
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user_id: %w", err)
	}
	return s.favoritesRepo.Remove(ctx, uid, vacancyID)
}

func (s *Service) ListFavorites(ctx context.Context, userID string) ([]*client.HHVacancy, error) {
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	ids, err := s.favoritesRepo.ListIDs(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to list favorite IDs: %w", err)
	}

	vacancies := make([]*client.HHVacancy, 0, len(ids))
	for _, id := range ids {
		v, err := s.hhClient.GetVacancyByID(ctx, id)
		if err != nil {
			continue
		}
		vacancies = append(vacancies, v)
	}
	return vacancies, nil
}

func (s *Service) GetFavoriteIDs(ctx context.Context, userID string) ([]string, error) {
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}
	return s.favoritesRepo.ListIDs(ctx, uid)
}

func (s *Service) DeleteUserData(ctx context.Context, userID string) error {
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user_id: %w", err)
	}
	return s.favoritesRepo.DeleteByUser(ctx, uid)
}
