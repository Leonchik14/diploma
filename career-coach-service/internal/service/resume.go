package service

import (
	"context"
	"fmt"

	"career-coach-service/internal/client"
	"career-coach-service/internal/extractor"
	"career-coach-service/internal/model"
	"career-coach-service/internal/parser"
	"career-coach-service/internal/repository"

	pbuser "proto/user"
)

const confidenceThreshold = 0.85
const maxQuestionsPerRound = 3

type ResumeService struct {
	parser          *parser.Parser
	repo            *repository.Repository
	extractor       *extractor.Extractor
	materialsClient *client.MaterialsClient
	userClient      *client.UserClient
}

func NewResumeService(parser *parser.Parser, repo *repository.Repository, extractor *extractor.Extractor, materialsClient *client.MaterialsClient, userClient *client.UserClient) *ResumeService {
	return &ResumeService{
		parser:          parser,
		repo:            repo,
		extractor:       extractor,
		materialsClient: materialsClient,
		userClient:      userClient,
	}
}

func (s *ResumeService) ParseResume(ctx context.Context, userID uint, materialID string) (*model.ResumeParseResponse, error) {
	fileStream, mimeType, err := s.materialsClient.DownloadFile(ctx, materialID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer fileStream.Close()

	text, err := s.extractor.ExtractText(ctx, fileStream, mimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text: %w", err)
	}

	draft, questions, err := s.parser.ParseResume(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resume: %w", err)
	}

	if draft.Confidence == nil {
		draft.Confidence = make(map[string]float64)
	}

	// Build confirmed_fields: true where confidence >= threshold
	confirmedFields := make(map[string]bool)
	confidenceMap := make(map[string]float64)
	for k, v := range draft.Confidence {
		confidenceMap[k] = v
		if v >= confidenceThreshold {
			confirmedFields[k] = true
		}
	}

	pbProfile := draftToProtoProfile(draft)
	version, err := s.userClient.UpsertResumeProfileInternal(ctx, userID, materialID, pbProfile, pbuser.ResumeProfileStatus_DRAFT, confirmedFields, confidenceMap)
	if err != nil {
		return nil, fmt.Errorf("failed to save profile: %w", err)
	}

	sessionID, err := s.repo.CreateResumeSession(ctx, userID, materialID, version, questions)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &model.ResumeParseResponse{
		SessionID: sessionID,
		Draft:     draft,
		Questions: questions,
		Status:    "awaiting_user",
	}, nil
}

func (s *ResumeService) AnswerQuestions(ctx context.Context, userID uint, req *model.ResumeAnswerRequest) (*model.ResumeAnswerResponse, error) {
	row, err := s.repo.GetResumeSession(ctx, req.SessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	getResp, err := s.userClient.GetResumeProfileInternal(ctx, userID)
	if err != nil || getResp == nil || getResp.Profile == nil {
		return nil, fmt.Errorf("failed to load profile: %w", err)
	}

	draft := protoProfileToDraft(getResp)
	updatedDraft, _, status := s.parser.ApplyAnswers(draft, row.Questions, req.Answers)

	// Build patch (only changed fields) and set_confirmed_fields from answers
	patch := buildPatchFromDraft(updatedDraft)
	setConfirmed := make(map[string]bool)
	setConfidence := make(map[string]float64)
	for _, a := range req.Answers {
		if a.QuestionID != "" {
			setConfirmed[a.QuestionID] = true
			setConfidence[a.QuestionID] = 1.0
		}
	}

	var statusProto *pbuser.ResumeProfileStatus
	if status == "completed" {
		s := pbuser.ResumeProfileStatus_CONFIRMED
		statusProto = &s
	}

	version, err := s.userClient.PatchResumeProfileInternal(ctx, userID, patch, setConfirmed, setConfidence, statusProto)
	if err != nil {
		return nil, fmt.Errorf("failed to patch profile: %w", err)
	}

	// Next questions: low confidence in updated draft, max 3
	nextQuestions := nextQuestionsFromDraft(updatedDraft, maxQuestionsPerRound)
	if status == "completed" {
		nextQuestions = nil
	}

	if err := s.repo.UpdateResumeSession(ctx, req.SessionID, userID, nextQuestions, status, version); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return &model.ResumeAnswerResponse{
		SessionID: req.SessionID,
		Draft:     updatedDraft,
		Questions: nextQuestions,
		Status:    status,
	}, nil
}

func (s *ResumeService) GetSession(ctx context.Context, sessionID string, userID uint) (*model.ResumeSessionResponse, error) {
	row, err := s.repo.GetResumeSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	getResp, err := s.userClient.GetResumeProfileInternal(ctx, userID)
	if err != nil || getResp == nil {
		return &model.ResumeSessionResponse{
			SessionID: row.SessionID,
			Draft:     nil,
			Questions: row.Questions,
			Status:    row.Status,
		}, nil
	}
	draft := protoProfileToDraft(getResp)
	return &model.ResumeSessionResponse{
		SessionID: row.SessionID,
		Draft:     draft,
		Questions: row.Questions,
		Status:    row.Status,
	}, nil
}

func draftToProtoProfile(d *model.ResumeProfileDraft) *pbuser.ResumeProfile {
	if d == nil {
		return nil
	}
	targetRoles := d.TargetRoles
	if targetRoles == nil {
		targetRoles = []string{}
	}
	workFormat := d.WorkFormat
	if workFormat == nil {
		workFormat = []string{}
	}
	skillsTop := d.SkillsTop
	if skillsTop == nil {
		skillsTop = []string{}
	}
	p := &pbuser.ResumeProfile{
		TargetRoles: targetRoles,
		WorkFormat:  workFormat,
		SkillsTop:   skillsTop,
	}
	if d.ExperienceLevel != nil {
		p.ExperienceLevel = d.ExperienceLevel
	}
	if d.SalaryMin != nil {
		p.SalaryMin = d.SalaryMin
	}
	if d.Currency != nil {
		p.Currency = d.Currency
	}
	if d.Notes != nil {
		p.Notes = d.Notes
	}
	for _, a := range d.Areas {
		p.Areas = append(p.Areas, &pbuser.Area{Id: a.ID, Name: a.Name})
	}
	return p
}

func protoProfileToDraft(resp *pbuser.GetResumeProfileInternalResponse) *model.ResumeProfileDraft {
	if resp == nil || resp.Profile == nil {
		return &model.ResumeProfileDraft{Confidence: make(map[string]float64)}
	}
	p := resp.Profile
	d := &model.ResumeProfileDraft{
		TargetRoles: p.TargetRoles,
		WorkFormat:  p.WorkFormat,
		SkillsTop:   p.SkillsTop,
		Confidence:  make(map[string]float64),
	}
	if p.ExperienceLevel != nil {
		d.ExperienceLevel = p.ExperienceLevel
	}
	if p.SalaryMin != nil {
		d.SalaryMin = p.SalaryMin
	}
	if p.Currency != nil {
		d.Currency = p.Currency
	}
	if p.Notes != nil {
		d.Notes = p.Notes
	}
	for _, a := range p.Areas {
		d.Areas = append(d.Areas, model.Area{ID: a.Id, Name: a.Name})
	}
	for k, v := range resp.Confidence {
		d.Confidence[k] = v
	}
	return d
}

func buildPatchFromDraft(d *model.ResumeProfileDraft) *pbuser.ResumeProfilePatch {
	if d == nil {
		return nil
	}
	targetRoles := d.TargetRoles
	if targetRoles == nil {
		targetRoles = []string{}
	}
	workFormat := d.WorkFormat
	if workFormat == nil {
		workFormat = []string{}
	}
	skillsTop := d.SkillsTop
	if skillsTop == nil {
		skillsTop = []string{}
	}
	p := &pbuser.ResumeProfilePatch{
		TargetRoles: targetRoles,
		WorkFormat:  workFormat,
		SkillsTop:   skillsTop,
	}
	if d.ExperienceLevel != nil {
		p.ExperienceLevel = d.ExperienceLevel
	}
	if d.SalaryMin != nil {
		p.SalaryMin = d.SalaryMin
	}
	if d.Currency != nil {
		p.Currency = d.Currency
	}
	if d.Notes != nil {
		p.Notes = d.Notes
	}
	for _, a := range d.Areas {
		p.Areas = append(p.Areas, &pbuser.Area{Id: a.ID, Name: a.Name})
	}
	return p
}

func nextQuestionsFromDraft(draft *model.ResumeProfileDraft, maxN int) []model.Question {
	if draft == nil || draft.Confidence == nil {
		return nil
	}
	var out []model.Question
	fieldIDs := []string{"target_roles", "experience_level", "areas", "salary_min", "work_format"}
	for _, id := range fieldIDs {
		if len(out) >= maxN {
			break
		}
		conf := draft.Confidence[id]
		if conf >= 0.6 {
			continue
		}
		out = append(out, model.Question{
			ID:   id,
			Text: "Please confirm or correct: " + id,
			Type: "free_text",
		})
	}
	return out
}
