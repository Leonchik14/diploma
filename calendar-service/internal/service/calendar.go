package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"calendar-service/internal/model"
	"calendar-service/internal/repo/postgres"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CalendarService struct {
	repo *postgres.EventsRepo
}

const (
	maxTitleLen       = 200
	maxDescriptionLen = 5000
	maxLocationLen    = 255
)

func NewCalendarService(repo *postgres.EventsRepo) *CalendarService {
	return &CalendarService{repo: repo}
}

func (s *CalendarService) CreateEvent(ctx context.Context, userID uint, event *model.Event) (*model.Event, error) {
	event.Title = strings.TrimSpace(event.Title)
	if event.Title == "" {
		return nil, status.Errorf(codes.InvalidArgument, "title is required")
	}
	if utf8.RuneCountInString(event.Title) > maxTitleLen {
		return nil, status.Errorf(codes.InvalidArgument, "title is too long (max %d characters)", maxTitleLen)
	}
	if event.Description != nil && utf8.RuneCountInString(*event.Description) > maxDescriptionLen {
		return nil, status.Errorf(codes.InvalidArgument, "description is too long (max %d characters)", maxDescriptionLen)
	}
	if event.Location != nil && utf8.RuneCountInString(*event.Location) > maxLocationLen {
		return nil, status.Errorf(codes.InvalidArgument, "location is too long (max %d characters)", maxLocationLen)
	}
	if event.Timezone != nil {
		tz := strings.TrimSpace(*event.Timezone)
		if tz != "" {
			if _, err := time.LoadLocation(tz); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid timezone: expected IANA format")
			}
			event.Timezone = &tz
		}
	}
	if event.StartTime.After(event.EndTime) || event.StartTime.Equal(event.EndTime) {
		return nil, status.Errorf(codes.InvalidArgument, "start_time must be before end_time")
	}

	event.UserID = userID
	if err := s.repo.Create(ctx, event); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create event: %v", err)
	}

	return event, nil
}

func (s *CalendarService) GetEvent(ctx context.Context, userID uint, eventID string) (*model.Event, error) {
	event, err := s.repo.GetByID(ctx, userID, eventID)
	if err != nil {
		if err.Error() == "event not found" {
			return nil, status.Errorf(codes.NotFound, "event not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get event: %v", err)
	}
	return event, nil
}

func (s *CalendarService) UpdateEvent(ctx context.Context, userID uint, eventID string, patch *model.EventPatch) (*model.Event, error) {
	var updateErr error
	err := s.repo.Update(ctx, userID, eventID, func(event *model.Event) {
		if patch.Title != nil {
			title := strings.TrimSpace(*patch.Title)
			if title == "" {
				updateErr = status.Errorf(codes.InvalidArgument, "title cannot be empty")
				return
			}
			if utf8.RuneCountInString(title) > maxTitleLen {
				updateErr = status.Errorf(codes.InvalidArgument, "title is too long (max %d characters)", maxTitleLen)
				return
			}
			event.Title = title
		}
		if patch.Description != nil {
			if utf8.RuneCountInString(*patch.Description) > maxDescriptionLen {
				updateErr = status.Errorf(codes.InvalidArgument, "description is too long (max %d characters)", maxDescriptionLen)
				return
			}
			event.Description = patch.Description
		}
		if patch.EventType != nil {
			event.EventType = *patch.EventType
		}
		if patch.StartTime != nil {
			event.StartTime = *patch.StartTime
		}
		if patch.EndTime != nil {
			event.EndTime = *patch.EndTime
		}
		if patch.Timezone != nil {
			tz := strings.TrimSpace(*patch.Timezone)
			if tz != "" {
				if _, err := time.LoadLocation(tz); err != nil {
					updateErr = status.Errorf(codes.InvalidArgument, "invalid timezone: expected IANA format")
					return
				}
				event.Timezone = &tz
			} else {
				event.Timezone = patch.Timezone
			}
		}
		if patch.Location != nil {
			if utf8.RuneCountInString(*patch.Location) > maxLocationLen {
				updateErr = status.Errorf(codes.InvalidArgument, "location is too long (max %d characters)", maxLocationLen)
				return
			}
			event.Location = patch.Location
		}
		if patch.RelatedVacancyID != nil {
			event.RelatedVacancyID = patch.RelatedVacancyID
		}
		if patch.ReminderEnabled != nil && !*patch.ReminderEnabled {
			event.ReminderMinutes = 0
		}
		if patch.ReminderMinutes != nil {
			event.ReminderMinutes = *patch.ReminderMinutes
		}

		if event.StartTime.After(event.EndTime) || event.StartTime.Equal(event.EndTime) {
			updateErr = status.Errorf(codes.InvalidArgument, "start_time must be before end_time")
			return
		}
	})

	if updateErr != nil {
		return nil, updateErr
	}

	if err != nil {
		if err.Error() == "event not found" {
			return nil, status.Errorf(codes.NotFound, "event not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update event: %v", err)
	}

	return s.repo.GetByID(ctx, userID, eventID)
}

func (s *CalendarService) DeleteEvent(ctx context.Context, userID uint, eventID string) error {
	err := s.repo.Delete(ctx, userID, eventID)
	if err != nil {
		if err.Error() == "event not found" {
			return status.Errorf(codes.NotFound, "event not found")
		}
		return status.Errorf(codes.Internal, "failed to delete event: %v", err)
	}
	return nil
}

func (s *CalendarService) ListEvents(ctx context.Context, userID uint, fromTime, toTime time.Time, pageSize int32, pageToken string, sortAsc bool) ([]*model.Event, string, error) {
	if fromTime.After(toTime) {
		return nil, "", status.Errorf(codes.InvalidArgument, "from_time must be before to_time")
	}

	diff := toTime.Sub(fromTime)
	if diff > 365*24*time.Hour {
		return nil, "", status.Errorf(codes.InvalidArgument, "time range cannot exceed 1 year")
	}

	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	events, nextToken, err := s.repo.List(ctx, userID, fromTime, toTime, pageSize, pageToken, sortAsc)
	if err != nil {
		return nil, "", status.Errorf(codes.Internal, "failed to list events: %v", err)
	}

	return events, nextToken, nil
}

func (s *CalendarService) ListUpcoming(ctx context.Context, userID uint, fromTime time.Time, limit int32) ([]*model.Event, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	events, err := s.repo.ListUpcoming(ctx, userID, fromTime, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list upcoming events: %v", err)
	}

	return events, nil
}

func (s *CalendarService) GetInterviewStats(ctx context.Context, userID uint) (upcoming, completed, total int32, err error) {
	upcoming, completed, total, err = s.repo.CountInterviews(ctx, userID)
	if err != nil {
		return 0, 0, 0, status.Errorf(codes.Internal, "failed to get interview stats: %v", err)
	}
	return
}

func (s *CalendarService) DeleteUserData(ctx context.Context, userID uint) error {
	return s.repo.DeleteUserData(ctx, userID)
}
