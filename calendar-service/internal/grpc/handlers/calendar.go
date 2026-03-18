package handlers

import (
	"context"
	"log/slog"
	"time"

	"calendar-service/internal/model"
	"calendar-service/internal/requestctx"
	"calendar-service/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbcalendar "proto/calendar"
)

type CalendarHandler struct {
	pbcalendar.UnimplementedCalendarServiceServer
	svc    *service.CalendarService
	logger *slog.Logger
}

func NewCalendarHandler(svc *service.CalendarService, logger *slog.Logger) *CalendarHandler {
	return &CalendarHandler{
		svc:    svc,
		logger: logger,
	}
}

func (h *CalendarHandler) CreateEvent(ctx context.Context, req *pbcalendar.CreateEventRequest) (*pbcalendar.CreateEventResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}

	event, err := h.pbToModel(req.Event)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event: %v", err)
	}

	created, err := h.svc.CreateEvent(ctx, userID, event)
	if err != nil {
		return nil, err
	}

	return &pbcalendar.CreateEventResponse{
		Event: h.modelToPB(created),
	}, nil
}

func (h *CalendarHandler) GetEvent(ctx context.Context, req *pbcalendar.GetEventRequest) (*pbcalendar.GetEventResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}

	event, err := h.svc.GetEvent(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}

	return &pbcalendar.GetEventResponse{
		Event: h.modelToPB(event),
	}, nil
}

func (h *CalendarHandler) UpdateEvent(ctx context.Context, req *pbcalendar.UpdateEventRequest) (*pbcalendar.UpdateEventResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}

	patch, err := h.pbPatchToModel(req.Patch)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid patch: %v", err)
	}

	event, err := h.svc.UpdateEvent(ctx, userID, req.Id, patch)
	if err != nil {
		return nil, err
	}

	return &pbcalendar.UpdateEventResponse{
		Event: h.modelToPB(event),
	}, nil
}

func (h *CalendarHandler) DeleteEvent(ctx context.Context, req *pbcalendar.DeleteEventRequest) (*pbcalendar.DeleteEventResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}

	if err := h.svc.DeleteEvent(ctx, userID, req.Id); err != nil {
		return nil, err
	}

	return &pbcalendar.DeleteEventResponse{Success: true}, nil
}

func (h *CalendarHandler) ListEvents(ctx context.Context, req *pbcalendar.ListEventsRequest) (*pbcalendar.ListEventsResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}

	if req.FromTime == nil || req.ToTime == nil {
		return nil, status.Errorf(codes.InvalidArgument, "from_time and to_time are required")
	}

	fromTime := req.FromTime.AsTime()
	toTime := req.ToTime.AsTime()
	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 50
	}
	sortAsc := req.Sort == pbcalendar.SortOrder_SORT_START_ASC

	events, nextToken, err := h.svc.ListEvents(ctx, userID, fromTime, toTime, pageSize, req.PageToken, sortAsc)
	if err != nil {
		return nil, err
	}

	pbEvents := make([]*pbcalendar.Event, len(events))
	for i, e := range events {
		pbEvents[i] = h.modelToPB(e)
	}

	return &pbcalendar.ListEventsResponse{
		Events:       pbEvents,
		NextPageToken: nextToken,
	}, nil
}

func (h *CalendarHandler) ListUpcoming(ctx context.Context, req *pbcalendar.ListUpcomingRequest) (*pbcalendar.ListUpcomingResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}

	fromTime := time.Now()
	if req.FromTime != nil {
		fromTime = req.FromTime.AsTime()
	}

	limit := req.Limit
	if limit == 0 {
		limit = 10
	}

	events, err := h.svc.ListUpcoming(ctx, userID, fromTime, limit)
	if err != nil {
		return nil, err
	}

	pbEvents := make([]*pbcalendar.Event, len(events))
	for i, e := range events {
		pbEvents[i] = h.modelToPB(e)
	}

	return &pbcalendar.ListUpcomingResponse{
		Events: pbEvents,
	}, nil
}

func (h *CalendarHandler) GetInterviewStats(ctx context.Context, req *pbcalendar.GetInterviewStatsRequest) (*pbcalendar.GetInterviewStatsResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}

	upcoming, total, err := h.svc.GetInterviewStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &pbcalendar.GetInterviewStatsResponse{
		Upcoming: upcoming,
		Total:    total,
	}, nil
}

func (h *CalendarHandler) DeleteUserData(ctx context.Context, req *pbcalendar.DeleteUserDataRequest) (*pbcalendar.DeleteUserDataResponse, error) {
	userID := uint(req.UserId)

	if err := h.svc.DeleteUserData(ctx, userID); err != nil {
		h.logger.Error("failed to delete user data", "user_id", userID, "error", err)
		return &pbcalendar.DeleteUserDataResponse{Ok: true}, nil
	}

	return &pbcalendar.DeleteUserDataResponse{Ok: true}, nil
}

func (h *CalendarHandler) pbToModel(pb *pbcalendar.Event) (*model.Event, error) {
	if pb == nil {
		return nil, status.Errorf(codes.InvalidArgument, "event is required")
	}

	reminderMinutes := int32(0)
	if pb.ReminderEnabled {
		reminderMinutes = pb.ReminderMinutes
	}

	e := &model.Event{
		Title:           pb.Title,
		EventType:       model.EventType(pb.EventType),
		ReminderMinutes: reminderMinutes,
	}

	if pb.Description != nil {
		e.Description = pb.Description
	}
	if pb.Timezone != nil {
		e.Timezone = pb.Timezone
	}
	if pb.Location != nil {
		e.Location = pb.Location
	}
	if pb.RelatedVacancyId != nil {
		e.RelatedVacancyID = pb.RelatedVacancyId
	}

	if pb.StartTime == nil {
		return nil, status.Errorf(codes.InvalidArgument, "start_time is required")
	}
	e.StartTime = pb.StartTime.AsTime()

	if pb.EndTime == nil {
		return nil, status.Errorf(codes.InvalidArgument, "end_time is required")
	}
	e.EndTime = pb.EndTime.AsTime()

	return e, nil
}

func (h *CalendarHandler) pbPatchToModel(pb *pbcalendar.EventPatch) (*model.EventPatch, error) {
	if pb == nil {
		return nil, status.Errorf(codes.InvalidArgument, "patch is required")
	}

	patch := &model.EventPatch{}

	if pb.Title != nil {
		patch.Title = pb.Title
	}
	if pb.Description != nil {
		patch.Description = pb.Description
	}
	if pb.EventType != nil {
		et := model.EventType(*pb.EventType)
		patch.EventType = &et
	}
	if pb.StartTime != nil {
		t := pb.StartTime.AsTime()
		patch.StartTime = &t
	}
	if pb.EndTime != nil {
		t := pb.EndTime.AsTime()
		patch.EndTime = &t
	}
	if pb.Timezone != nil {
		patch.Timezone = pb.Timezone
	}
	if pb.Location != nil {
		patch.Location = pb.Location
	}
	if pb.RelatedVacancyId != nil {
		patch.RelatedVacancyID = pb.RelatedVacancyId
	}
	if pb.ReminderEnabled != nil {
		patch.ReminderEnabled = pb.ReminderEnabled
	}
	if pb.ReminderMinutes != nil {
		patch.ReminderMinutes = pb.ReminderMinutes
	}

	return patch, nil
}

func (h *CalendarHandler) modelToPB(e *model.Event) *pbcalendar.Event {
	pb := &pbcalendar.Event{
		Id:                e.ID,
		Title:             e.Title,
		EventType:         pbcalendar.EventType(e.EventType),
		StartTime:         timestamppb.New(e.StartTime),
		EndTime:           timestamppb.New(e.EndTime),
		ReminderEnabled:   e.ReminderMinutes > 0,
		ReminderMinutes:   e.ReminderMinutes,
		CreatedAt:         timestamppb.New(e.CreatedAt),
		UpdatedAt:         timestamppb.New(e.UpdatedAt),
		Completed:         time.Now().After(e.EndTime),
	}

	if e.Description != nil {
		pb.Description = e.Description
	}
	if e.Timezone != nil {
		pb.Timezone = e.Timezone
	}
	if e.Location != nil {
		pb.Location = e.Location
	}
	if e.RelatedVacancyID != nil {
		pb.RelatedVacancyId = e.RelatedVacancyID
	}

	return pb
}
