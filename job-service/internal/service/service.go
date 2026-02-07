package service

import (
	"context"
	"errors"
	"fmt"
	"job-service/internal/client"
)

var (
	ErrResumeProfileIncomplete = errors.New("resume profile incomplete")
	ErrResumeProfileUnavailable = errors.New("resume profile not available")
)

type Service struct {
	userClient *client.UserClient
	hhClient   *client.HHClient
}

func NewService(userClient *client.UserClient, hhClient *client.HHClient) *Service {
	return &Service{
		userClient: userClient,
		hhClient:   hhClient,
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

func (s *Service) AddFavorite(ctx context.Context, userID string, vacancyID string) error {
	// TODO: Implement favorite jobs storage (database)
	return nil
}

func (s *Service) RemoveFavorite(ctx context.Context, userID string, vacancyID string) error {
	// TODO: Implement favorite jobs storage (database)
	return nil
}

func (s *Service) ListFavorites(ctx context.Context, userID string) ([]*client.HHVacancy, error) {
	// TODO: Implement favorite jobs storage (database)
	return []*client.HHVacancy{}, nil
}

// GetFavoriteIDs возвращает ID избранных вакансий пользователя (для проставления is_favorite в списке).
func (s *Service) GetFavoriteIDs(ctx context.Context, userID string) ([]string, error) {
	// TODO: когда будет БД избранного — читать оттуда
	return nil, nil
}

func (s *Service) DeleteUserData(ctx context.Context, userID string) error {
	return nil
}
