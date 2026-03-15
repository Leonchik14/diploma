package handlers

import (
	"context"
	"log/slog"

	"career-coach-service/internal/model"
	"career-coach-service/internal/requestctx"
	"career-coach-service/internal/service"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbcoach "proto/coach"
	pbuser "proto/user"
)

type CoachHandler struct {
	pbcoach.UnimplementedCoachServiceServer
	coachService  *service.CoachService
	resumeService *service.ResumeService
	logger        *slog.Logger
}

func NewCoachHandler(coachService *service.CoachService, resumeService *service.ResumeService, logger *slog.Logger) *CoachHandler {
	return &CoachHandler{
		coachService:  coachService,
		resumeService: resumeService,
		logger:        logger,
	}
}

func (h *CoachHandler) getUserID(ctx context.Context) (uint, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return 0, status.Errorf(codes.Unauthenticated, "user ID not found in context")
	}
	return userID, nil
}

func (h *CoachHandler) Ask(ctx context.Context, req *pbcoach.AskRequest) (*pbcoach.AskResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.Question == "" {
		return nil, status.Errorf(codes.InvalidArgument, "question is required")
	}

	askReq := &model.AskRequest{
		Question: req.Question,
	}

	// Handle optional conversation_id
	if req.ConversationId != nil {
		askReq.ConversationID = *req.ConversationId
	}

	// Convert proto ResumeProfile to model ResumeProfile
	if req.ResumeProfile != nil {
		askReq.ResumeProfile = convertProtoResumeProfile(req.ResumeProfile)
	}

	// Convert proto ContextChunks to model ContextChunks
	if len(req.ContextChunks) > 0 {
		askReq.ContextChunks = make([]model.ContextChunk, len(req.ContextChunks))
		for i, chunk := range req.ContextChunks {
			askReq.ContextChunks[i] = model.ContextChunk{
				Source:  chunk.Source,
				Title:   chunk.Title,
				Content: chunk.Content,
			}
		}
	}

	conversationIDStr := ""
	if req.ConversationId != nil {
		conversationIDStr = *req.ConversationId
	}
	h.logger.Info("processing ask request", "user_id", userID, "conversation_id", conversationIDStr)

	resp, err := h.coachService.Ask(ctx, userID, askReq)
	if err != nil {
		h.logger.Error("failed to process ask request", "error", err, "user_id", userID)
		return nil, status.Errorf(codes.Internal, "failed to process ask request: %v", err)
	}

	return &pbcoach.AskResponse{
		ConversationId: resp.ConversationID,
		Answer:         resp.Answer,
	}, nil
}

func (h *CoachHandler) ParseResume(ctx context.Context, req *pbcoach.ParseResumeRequest) (*pbcoach.ParseResumeResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.MaterialId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "material_id is required")
	}

	resp, err := h.resumeService.ParseResume(ctx, userID, req.MaterialId)
	if err != nil {
		h.logger.Error("failed to parse resume", "error", err, "user_id", userID, "material_id", req.MaterialId)
		// Return more detailed error message
		return nil, status.Errorf(codes.Internal, "failed to parse resume: %v", err)
	}

	return convertResumeParseResponse(resp), nil
}

func (h *CoachHandler) UploadAndParseResume(ctx context.Context, req *pbcoach.UploadAndParseResumeRequest) (*pbcoach.UploadAndParseResumeResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.FileContent) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "file_content is required")
	}
	if req.Filename == "" {
		return nil, status.Errorf(codes.InvalidArgument, "filename is required")
	}

	resp, err := h.resumeService.UploadAndParseResume(ctx, userID, req.FileContent, req.Filename)
	if err != nil {
		h.logger.Error("failed to upload and parse resume", "error", err, "user_id", userID)
		return nil, status.Errorf(codes.Internal, "failed to upload and parse resume: %v", err)
	}

	return &pbcoach.UploadAndParseResumeResponse{
		SessionId: resp.SessionID,
		Draft:     convertResumeProfileDraft(resp.Draft),
		Questions: convertQuestions(resp.Questions),
		Status:    resp.Status,
	}, nil
}

func (h *CoachHandler) AnswerResume(ctx context.Context, req *pbcoach.AnswerResumeRequest) (*pbcoach.AnswerResumeResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}
	if req.SessionId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "session_id is required")
	}
	if _, err := uuid.Parse(req.SessionId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "session_id must be a valid UUID")
	}

	answers := make([]model.QuestionAnswer, len(req.Answers))
	for i, a := range req.Answers {
		answers[i] = model.QuestionAnswer{
			QuestionID: a.QuestionId,
			Value:      a.Value,
		}
	}

	answerReq := &model.ResumeAnswerRequest{
		SessionID: req.SessionId,
		Answers:   answers,
	}

	resp, err := h.resumeService.AnswerQuestions(ctx, userID, answerReq)
	if err != nil {
		h.logger.Error("failed to answer questions", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to process answers")
	}

	return convertResumeAnswerResponse(resp), nil
}

func (h *CoachHandler) GetResumeSession(ctx context.Context, req *pbcoach.GetResumeSessionRequest) (*pbcoach.GetResumeSessionResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}
	if req.SessionId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "session_id is required")
	}
	if _, err := uuid.Parse(req.SessionId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "session_id must be a valid UUID")
	}

	resp, err := h.resumeService.GetSession(ctx, req.SessionId, userID)
	if err != nil {
		h.logger.Error("failed to get session", "error", err)
		return nil, status.Errorf(codes.NotFound, "session not found")
	}

	return convertResumeSessionResponse(resp), nil
}

func (h *CoachHandler) PrepareForVacancy(ctx context.Context, req *pbcoach.PrepareForVacancyRequest) (*pbcoach.PrepareForVacancyResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}
	if req.VacancyId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "vacancy_id is required")
	}

	recommendations, err := h.coachService.PrepareForVacancy(ctx, userID, req.VacancyId)
	if err != nil {
		h.logger.Error("failed to prepare for vacancy", "error", err, "user_id", userID, "vacancy_id", req.VacancyId)
		return nil, status.Errorf(codes.Internal, "failed to get recommendations: %v", err)
	}

	return &pbcoach.PrepareForVacancyResponse{
		Recommendations: recommendations,
	}, nil
}

func (h *CoachHandler) ReviewResume(ctx context.Context, req *pbcoach.ReviewResumeRequest) (*pbcoach.ReviewResumeResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	score, recommendations, err := h.coachService.ReviewResume(ctx, userID)
	if err != nil {
		h.logger.Error("failed to review resume", "error", err, "user_id", userID)
		return nil, status.Errorf(codes.Internal, "failed to review resume: %v", err)
	}

	return &pbcoach.ReviewResumeResponse{
		Score:          score,
		Recommendations: recommendations,
	}, nil
}

func (h *CoachHandler) DeleteUserData(ctx context.Context, req *pbcoach.DeleteUserDataRequest) (*pbcoach.DeleteUserDataResponse, error) {
	userID := uint(req.UserId)

	if err := h.coachService.DeleteUserData(ctx, userID); err != nil {
		h.logger.Error("failed to delete user data", "user_id", userID, "error", err)
		return &pbcoach.DeleteUserDataResponse{Ok: true}, nil
	}

	return &pbcoach.DeleteUserDataResponse{Ok: true}, nil
}

// Helper functions to convert internal models to proto messages
func convertResumeParseResponse(resp *model.ResumeParseResponse) *pbcoach.ParseResumeResponse {
	return &pbcoach.ParseResumeResponse{
		SessionId: resp.SessionID,
		Draft:     convertResumeProfileDraft(resp.Draft),
		Questions: convertQuestions(resp.Questions),
		Status:    resp.Status,
	}
}

func convertResumeAnswerResponse(resp *model.ResumeAnswerResponse) *pbcoach.AnswerResumeResponse {
	return &pbcoach.AnswerResumeResponse{
		SessionId: resp.SessionID,
		Draft:     convertResumeProfileDraft(resp.Draft),
		Questions: convertQuestions(resp.Questions),
		Status:    resp.Status,
	}
}

func convertResumeSessionResponse(resp *model.ResumeSessionResponse) *pbcoach.GetResumeSessionResponse {
	return &pbcoach.GetResumeSessionResponse{
		SessionId: resp.SessionID,
		Draft:     convertResumeProfileDraft(resp.Draft),
		Questions: convertQuestions(resp.Questions),
		Status:    resp.Status,
	}
}

func convertResumeProfileDraft(draft *model.ResumeProfileDraft) *pbcoach.ResumeProfileDraft {
	if draft == nil {
		return nil
	}
	targetRoles := draft.TargetRoles
	if targetRoles == nil {
		targetRoles = []string{}
	}

	pbDraft := &pbcoach.ResumeProfileDraft{
		TargetRoles:     targetRoles,
		ExperienceLevel: draft.ExperienceLevel,
		SalaryMin:       draft.SalaryMin,
		Currency:        draft.Currency,
		WorkFormat:      draft.WorkFormat,
		SkillsTop:       draft.SkillsTop,
		Notes:           draft.Notes,
		Confidence:      draft.Confidence,
	}

	// Convert ProfessionalRoleCandidates
	for _, prc := range draft.ProfessionalRoleCandidates {
		pbDraft.ProfessionalRoleCandidates = append(pbDraft.ProfessionalRoleCandidates, &pbcoach.ProfessionalRoleCandidate{
			Id:         prc.ID,
			Name:       prc.Name,
			Confidence: prc.Confidence,
		})
	}

	// Convert Areas
	for _, area := range draft.Areas {
		pbDraft.Areas = append(pbDraft.Areas, &pbcoach.Area{
			Id:         area.ID,
			Name:       area.Name,
			Confidence: area.Confidence,
		})
	}

	return pbDraft
}

func convertQuestions(questions []model.Question) []*pbcoach.Question {
	pbQuestions := make([]*pbcoach.Question, len(questions))
	for i, q := range questions {
		pbQuestions[i] = &pbcoach.Question{
			Id:      q.ID,
			Text:    q.Text,
			Type:    q.Type,
			Options: q.Options,
		}
	}
	return pbQuestions
}

func convertProtoResumeProfile(pbProfile *pbuser.ResumeProfile) *model.ResumeProfile {
	if pbProfile == nil {
		return nil
	}

	profile := &model.ResumeProfile{}

	// Convert target_roles to Role (take first role if available)
	if len(pbProfile.TargetRoles) > 0 {
		profile.Role = pbProfile.TargetRoles[0]
	}

	// Convert experience_level to Experience
	if pbProfile.ExperienceLevel != nil {
		profile.Experience = *pbProfile.ExperienceLevel
	}

	// Convert skills_top to Skills
	profile.Skills = pbProfile.SkillsTop

	// Convert salary_min to SalaryExpectation
	if pbProfile.SalaryMin != nil {
		profile.SalaryExpectation = pbProfile.SalaryMin
	}

	return profile
}
