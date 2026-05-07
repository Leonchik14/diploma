package handlers

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

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

const (
	maxQuestionLen         = 2000
	maxContextChunks       = 20
	maxContextChunkContent = 4000
	maxChatMessageLen      = 4000
)

func hasUnsafeControlChars(s string) bool {
	for _, r := range s {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return true
		}
	}
	return false
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

	question := strings.TrimSpace(req.Question)
	if question == "" {
		return nil, status.Errorf(codes.InvalidArgument, "question is required")
	}
	if utf8.RuneCountInString(question) > maxQuestionLen {
		return nil, status.Errorf(codes.InvalidArgument, "question is too long (max %d characters)", maxQuestionLen)
	}
	if hasUnsafeControlChars(question) {
		return nil, status.Errorf(codes.InvalidArgument, "question contains unsupported control characters")
	}
	if len(req.ContextChunks) > maxContextChunks {
		return nil, status.Errorf(codes.InvalidArgument, "too many context chunks (max %d)", maxContextChunks)
	}

	askReq := &model.AskRequest{
		Question: question,
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
			content := strings.TrimSpace(chunk.Content)
			if content == "" {
				return nil, status.Errorf(codes.InvalidArgument, "context chunk content is required")
			}
			if utf8.RuneCountInString(content) > maxContextChunkContent {
				return nil, status.Errorf(codes.InvalidArgument, "context chunk content is too long (max %d characters)", maxContextChunkContent)
			}
			if hasUnsafeControlChars(content) {
				return nil, status.Errorf(codes.InvalidArgument, "context chunk contains unsupported control characters")
			}
			askReq.ContextChunks[i] = model.ContextChunk{
				Source:  chunk.Source,
				Title:   chunk.Title,
				Content: content,
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
		if strings.Contains(err.Error(), "invalid answer:") {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
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
		if errors.Is(err, service.ErrResumeNotUploaded) {
			return nil, status.Errorf(codes.FailedPrecondition, "resume is not uploaded")
		}
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
		if errors.Is(err, service.ErrResumeNotUploaded) {
			return nil, status.Errorf(codes.FailedPrecondition, "resume is not uploaded")
		}
		h.logger.Error("failed to review resume", "error", err, "user_id", userID)
		return nil, status.Errorf(codes.Internal, "failed to review resume: %v", err)
	}

	return &pbcoach.ReviewResumeResponse{
		Score:           score,
		Recommendations: recommendations,
	}, nil
}

func (h *CoachHandler) AddChatMessage(ctx context.Context, req *pbcoach.AddChatMessageRequest) (*pbcoach.AddChatMessageResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, status.Errorf(codes.InvalidArgument, "content is required")
	}
	if utf8.RuneCountInString(content) > maxChatMessageLen {
		return nil, status.Errorf(codes.InvalidArgument, "content is too long (max %d characters)", maxChatMessageLen)
	}
	if hasUnsafeControlChars(content) {
		return nil, status.Errorf(codes.InvalidArgument, "content contains unsupported control characters")
	}

	role := ""
	switch req.Owner {
	case pbcoach.ChatMessageOwner_CHAT_MESSAGE_OWNER_USER:
		role = "user"
	case pbcoach.ChatMessageOwner_CHAT_MESSAGE_OWNER_ASSISTANT:
		role = "assistant"
	default:
		return nil, status.Errorf(codes.InvalidArgument, "owner must be user or assistant")
	}

	addReq := &model.AddChatMessageRequest{
		Content: content,
		Role:    role,
	}
	if req.ConversationId != nil {
		addReq.ConversationID = strings.TrimSpace(*req.ConversationId)
	}

	resp, err := h.coachService.AddChatMessage(ctx, userID, addReq)
	if err != nil {
		h.logger.Error("failed to add chat message", "error", err, "user_id", userID)
		return nil, status.Errorf(codes.Internal, "failed to add chat message")
	}

	return &pbcoach.AddChatMessageResponse{
		ConversationId: resp.ConversationID,
	}, nil
}

func (h *CoachHandler) GetCoachChatHistory(ctx context.Context, req *pbcoach.GetCoachChatHistoryRequest) (*pbcoach.GetCoachChatHistoryResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}
	entries, total, err := h.coachService.GetCoachChatHistory(ctx, userID, int(req.GetPageSize()), int(req.GetPageOffset()))
	if err != nil {
		h.logger.Error("get coach chat history", "error", err, "user_id", userID)
		return nil, status.Errorf(codes.Internal, "failed to load chat history")
	}
	out := make([]*pbcoach.CoachHistoryEntry, 0, len(entries))
	for _, e := range entries {
		pb := &pbcoach.CoachHistoryEntry{
			Kind:           coachHistoryKindToProto(e.Kind),
			ConversationId: e.ConversationID,
			Content:        e.Content,
			CreatedAt:      e.CreatedAt.UTC().Format(time.RFC3339),
		}
		if e.ResumeScore != nil {
			pb.ResumeScore = e.ResumeScore
		}
		if e.VacancyID != "" {
			v := e.VacancyID
			pb.VacancyId = &v
		}
		out = append(out, pb)
	}
	return &pbcoach.GetCoachChatHistoryResponse{
		Entries:    out,
		TotalCount: int32(total),
	}, nil
}

func coachHistoryKindToProto(k model.CoachHistoryKind) pbcoach.CoachHistoryEntryKind {
	switch k {
	case model.CoachHistoryAskUser:
		return pbcoach.CoachHistoryEntryKind_COACH_HISTORY_ENTRY_KIND_ASK_USER
	case model.CoachHistoryAskAssistant:
		return pbcoach.CoachHistoryEntryKind_COACH_HISTORY_ENTRY_KIND_ASK_ASSISTANT
	case model.CoachHistoryReviewResume:
		return pbcoach.CoachHistoryEntryKind_COACH_HISTORY_ENTRY_KIND_REVIEW_RESUME
	case model.CoachHistoryPrepareVacancy:
		return pbcoach.CoachHistoryEntryKind_COACH_HISTORY_ENTRY_KIND_PREPARE_VACANCY
	default:
		return pbcoach.CoachHistoryEntryKind_COACH_HISTORY_ENTRY_KIND_UNSPECIFIED
	}
}

func (h *CoachHandler) ClearChatHistory(ctx context.Context, req *pbcoach.ClearChatHistoryRequest) (*pbcoach.ClearChatHistoryResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}
	convID := ""
	if req.ConversationId != nil {
		convID = strings.TrimSpace(*req.ConversationId)
	}
	deleted, err := h.coachService.ClearChatHistory(ctx, userID, convID)
	if err != nil {
		h.logger.Error("clear chat history failed", "user_id", userID, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to clear chat history")
	}
	return &pbcoach.ClearChatHistoryResponse{
		Ok:                   true,
		DeletedConversations: int32(deleted),
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
		ExperienceLevel: humanReadableExperienceLevel(draft.ExperienceLevel),
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

func humanReadableExperienceLevel(level *string) *string {
	if level == nil {
		return nil
	}
	var out string
	switch *level {
	case "noExperience":
		out = "Нет опыта"
	case "between1And3":
		out = "1-3 года"
	case "between3And6":
		out = "3-6 лет"
	case "moreThan6":
		out = "6+ лет"
	default:
		out = *level
	}
	return &out
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
