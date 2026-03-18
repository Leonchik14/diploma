package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"user-service/internal/clients"
	"user-service/internal/config"
	"user-service/internal/database"
	"user-service/internal/email"
	"user-service/internal/repo/postgres"
	"user-service/internal/requestctx"
	"user-service/internal/service"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbuser "proto/user"
)

type UserHandler struct {
	pbuser.UnimplementedUserServiceServer
	cfg                *config.Config
	logger             *slog.Logger
	passwordResetSvc   *service.PasswordResetService
	accountDeletionSvc *service.AccountDeletionService
	resumeRepo         *postgres.ResumeProfileRepo
	materialsClient    *clients.MaterialsClient
	calendarClient     *clients.CalendarClient
}

func NewUserHandler(cfg *config.Config, logger *slog.Logger) *UserHandler {
	smtpPort, _ := strconv.Atoi(cfg.SMTPPort)
	sender := email.NewSMTPSender(
		cfg.SMTPHost, smtpPort, cfg.SMTPUser, cfg.SMTPPassword,
		cfg.SMTPFromEmail, cfg.SMTPFromName, cfg.SMTPTLS, logger)
	passwordResetSvc := service.NewPasswordResetService(cfg, sender)
	accountDeletionSvc := service.NewAccountDeletionService(cfg, logger)
	resumeRepo := postgres.NewResumeProfileRepo()
	materialsClient := clients.NewMaterialsClient(cfg.MaterialsServiceAddr, cfg.InternalAPIKey, logger)
	calendarClient := clients.NewCalendarClient(cfg.CalendarServiceAddr, cfg.InternalAPIKey, logger)

	return &UserHandler{
		cfg:                cfg,
		logger:             logger,
		passwordResetSvc:   passwordResetSvc,
		accountDeletionSvc: accountDeletionSvc,
		resumeRepo:         resumeRepo,
		materialsClient:    materialsClient,
		calendarClient:     calendarClient,
	}
}

func (h *UserHandler) GetMe(ctx context.Context, req *pbuser.GetMeRequest) (*pbuser.GetMeResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}

	var email, username, firstName, lastName string
	var deletedAt sql.NullTime
	var notificationsEnabled bool
	err := database.DB.QueryRow(ctx,
		"SELECT email, username, deleted_at, COALESCE(first_name,''), COALESCE(last_name,''), COALESCE(notifications_enabled, true) FROM users WHERE id = $1",
		userID).Scan(&email, &username, &deletedAt, &firstName, &lastName, &notificationsEnabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "database error")
	}

	if deletedAt.Valid {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	// resume_uploaded: считаем по наличию записи в таблице резюме
	resumeUploaded := false
	if h.resumeRepo != nil {
		if row, err := h.resumeRepo.GetByUserID(ctx, uint(userID)); err == nil && row != nil {
			resumeUploaded = true
		}
	}

	var totalIv, completedIv, upcomingIv int32
	if h.calendarClient != nil {
		upcomingIv, completedIv, totalIv, err = h.calendarClient.GetInterviewStats(ctx, uint(userID))
		if err != nil {
			h.logger.Warn("GetMe: calendar interview stats unavailable", "user_id", userID, "error", err)
			totalIv, completedIv, upcomingIv = 0, 0, 0
		}
	}

	return &pbuser.GetMeResponse{
		User: &pbuser.UserProfile{
			Id:                   uint32(userID),
			FirstName:            firstName,
			LastName:             lastName,
			Email:                email,
			Username:             username,
			ResumeUploaded:       resumeUploaded,
			TotalInterviews:      totalIv,
			CompletedInterviews:  completedIv,
			UpcomingInterviews:   upcomingIv,
			NotificationsEnabled: notificationsEnabled,
		},
	}, nil
}

func (h *UserHandler) UploadProfilePhoto(ctx context.Context, req *pbuser.UploadProfilePhotoRequest) (*pbuser.UploadProfilePhotoResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}
	if len(req.FileContent) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "file_content is required")
	}
	// Простая защита от слишком больших файлов (например, >5 МБ)
	if len(req.FileContent) > 5*1024*1024 {
		return nil, status.Errorf(codes.InvalidArgument, "file too large")
	}
	if h.materialsClient == nil {
		return nil, status.Errorf(codes.Internal, "materials client not configured")
	}

	filename := strings.TrimSpace(req.Filename)
	if filename == "" {
		filename = "profile_photo"
	}

	materialID, err := h.materialsClient.UploadUserProfilePhoto(ctx, uint(userID), req.FileContent, filename, req.MimeType)
	if err != nil {
		h.logger.Error("failed to upload profile photo to materials-service", "user_id", userID, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to upload profile photo")
	}

	_, err = database.DB.Exec(ctx,
		"UPDATE users SET profile_photo_material_id = $1, updated_at = NOW() WHERE id = $2",
		materialID, userID)
	if err != nil {
		h.logger.Error("failed to save profile photo material id", "user_id", userID, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to save profile photo")
	}

	return &pbuser.UploadProfilePhotoResponse{Ok: true}, nil
}

func (h *UserHandler) GetProfilePhoto(ctx context.Context, req *pbuser.GetProfilePhotoRequest) (*pbuser.GetProfilePhotoResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}
	if h.materialsClient == nil {
		return nil, status.Errorf(codes.Internal, "materials client not configured")
	}

	var materialID sql.NullString
	err := database.DB.QueryRow(ctx,
		"SELECT profile_photo_material_id FROM users WHERE id = $1",
		userID).Scan(&materialID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "database error")
	}
	if !materialID.Valid || strings.TrimSpace(materialID.String) == "" {
		return nil, status.Errorf(codes.NotFound, "profile photo not set")
	}

	resp, err := h.materialsClient.DownloadFile(ctx, materialID.String, uint(userID))
	if err != nil {
		h.logger.Error("failed to download profile photo from materials-service", "user_id", userID, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to load profile photo")
	}

	return &pbuser.GetProfilePhotoResponse{
		Content:  resp.Content,
		Filename: resp.Filename,
		MimeType: resp.MimeType,
	}, nil
}

func (h *UserHandler) GetResumeFile(ctx context.Context, req *pbuser.GetResumeFileRequest) (*pbuser.GetResumeFileResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}
	if h.materialsClient == nil {
		return nil, status.Errorf(codes.Internal, "materials client not configured")
	}

	row, err := h.resumeRepo.GetByUserID(ctx, uint(userID))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error")
	}
	if row == nil || row.SourceMaterialID == nil || *row.SourceMaterialID == "" {
		return nil, status.Errorf(codes.NotFound, "resume file not found")
	}

	resp, err := h.materialsClient.DownloadFile(ctx, *row.SourceMaterialID, uint(userID))
	if err != nil {
		h.logger.Error("failed to download resume file from materials-service", "user_id", userID, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to load resume file")
	}

	return &pbuser.GetResumeFileResponse{
		Content:  resp.Content,
		Filename: resp.Filename,
		MimeType: resp.MimeType,
	}, nil
}

func (h *UserHandler) GetResumeProfile(ctx context.Context, req *pbuser.GetResumeProfileRequest) (*pbuser.GetResumeProfileResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}
	return h.getResumeProfileResponse(ctx, uint(userID))
}

func (h *UserHandler) GetResumeProfileInternal(ctx context.Context, req *pbuser.GetResumeProfileInternalRequest) (*pbuser.GetResumeProfileInternalResponse, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id required")
	}
	resp, err := h.getResumeProfileResponse(ctx, uint(req.UserId))
	if err != nil {
		return nil, err
	}
	return &pbuser.GetResumeProfileInternalResponse{
		Profile:         resp.Profile,
		Status:          resp.Status,
		Version:         resp.Version,
		SourceMaterialId: resp.SourceMaterialId,
		ConfirmedFields: resp.ConfirmedFields,
		Confidence:      resp.Confidence,
	}, nil
}

func (h *UserHandler) getResumeProfileResponse(ctx context.Context, userID uint) (*pbuser.GetResumeProfileResponse, error) {
	row, err := h.resumeRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error")
	}
	if row == nil {
		return nil, status.Errorf(codes.NotFound, "resume profile not found")
	}

	pbProfile := rowToProtoProfile(row)
	statusProto := pbuser.ResumeProfileStatus_RESUME_PROFILE_STATUS_UNSPECIFIED
	if row.Status == "DRAFT" {
		statusProto = pbuser.ResumeProfileStatus_DRAFT
	} else if row.Status == "CONFIRMED" {
		statusProto = pbuser.ResumeProfileStatus_CONFIRMED
	}

	resp := &pbuser.GetResumeProfileResponse{
		Profile:  pbProfile,
		Status:   statusProto,
		Version:  row.Version,
		ConfirmedFields: row.ConfirmedFields,
		Confidence:      row.Confidence,
	}
	if row.SourceMaterialID != nil {
		resp.SourceMaterialId = row.SourceMaterialID
	}
	return resp, nil
}

func rowToProtoProfile(row *postgres.ResumeProfileRow) *pbuser.ResumeProfile {
	p := &pbuser.ResumeProfile{
		TargetRoles:  row.TargetRoles,
		WorkFormat:   row.WorkFormat,
		SkillsTop:    row.SkillsTop,
	}
	if row.ExperienceLevel != nil {
		p.ExperienceLevel = row.ExperienceLevel
	}
	if row.SalaryMin != nil {
		s := float64(*row.SalaryMin)
		p.SalaryMin = &s
	}
	if row.Currency != nil {
		p.Currency = row.Currency
	}
	if row.EducationLevel != nil {
		p.EducationLevel = row.EducationLevel
	}
	if row.Notes != nil {
		p.Notes = row.Notes
	}
	areas := row.AreaIDsToAreas()
	for i := range areas {
		p.Areas = append(p.Areas, &pbuser.Area{Id: areas[i].ID, Name: areas[i].Name})
	}
	return p
}

func (h *UserHandler) UpsertResumeProfileInternal(ctx context.Context, req *pbuser.UpsertResumeProfileInternalRequest) (*pbuser.UpsertResumeProfileInternalResponse, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id required")
	}
	if req.Profile == nil {
		return nil, status.Errorf(codes.InvalidArgument, "profile required")
	}

	statusStr := "DRAFT"
	if req.Status == pbuser.ResumeProfileStatus_CONFIRMED {
		statusStr = "CONFIRMED"
	}

	targetRoles := req.Profile.TargetRoles
	if targetRoles == nil {
		targetRoles = []string{}
	}
	areaIDs := make([]string, 0, len(req.Profile.Areas))
	for _, a := range req.Profile.Areas {
		areaIDs = append(areaIDs, a.Id)
	}
	workFormat := req.Profile.WorkFormat
	if workFormat == nil {
		workFormat = []string{}
	}
	skillsTop := req.Profile.SkillsTop
	if skillsTop == nil {
		skillsTop = []string{}
	}
	var salaryMin *int
	if req.Profile.SalaryMin != nil {
		v := int(*req.Profile.SalaryMin)
		salaryMin = &v
	}

	conf := map[string]interface{}{}
	for k, v := range req.Confidence {
		conf[k] = v
	}
	cf := map[string]interface{}{}
	for k, v := range req.ConfirmedFields {
		cf[k] = v
	}

	version, err := h.resumeRepo.Upsert(ctx, uint(req.UserId), statusStr, req.SourceMaterialId,
		targetRoles, req.Profile.ExperienceLevel, areaIDs, salaryMin, req.Profile.Currency,
		workFormat, skillsTop, req.Profile.EducationLevel, req.Profile.Notes,
		conf, cf)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert resume profile: %v", err)
	}
	return &pbuser.UpsertResumeProfileInternalResponse{Version: version}, nil
}

func (h *UserHandler) PatchResumeProfileInternal(ctx context.Context, req *pbuser.PatchResumeProfileInternalRequest) (*pbuser.PatchResumeProfileInternalResponse, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id required")
	}

	patch := make(map[string]interface{})
	if req.Patch != nil {
		if len(req.Patch.TargetRoles) > 0 {
			patch["target_roles"] = req.Patch.TargetRoles
		}
		if req.Patch.ExperienceLevel != nil {
			patch["experience_level"] = *req.Patch.ExperienceLevel
		}
		if len(req.Patch.Areas) > 0 {
			ids := make([]string, 0, len(req.Patch.Areas))
			for _, a := range req.Patch.Areas {
				ids = append(ids, a.Id)
			}
			patch["area_ids"] = ids
		}
		if req.Patch.SalaryMin != nil {
			patch["salary_min"] = int(*req.Patch.SalaryMin)
		}
		if req.Patch.Currency != nil {
			patch["currency"] = *req.Patch.Currency
		}
		if len(req.Patch.WorkFormat) > 0 {
			patch["work_format"] = req.Patch.WorkFormat
		}
		if len(req.Patch.SkillsTop) > 0 {
			patch["skills_top"] = req.Patch.SkillsTop
		}
		if req.Patch.EducationLevel != nil {
			patch["education_level"] = *req.Patch.EducationLevel
		}
		if req.Patch.Notes != nil {
			patch["notes"] = *req.Patch.Notes
		}
	}

	var setConfirmed, setConf map[string]interface{}
	if len(req.SetConfirmedFields) > 0 {
		setConfirmed = make(map[string]interface{})
		for k, v := range req.SetConfirmedFields {
			setConfirmed[k] = v
		}
	}
	if len(req.SetConfidence) > 0 {
		setConf = make(map[string]interface{})
		for k, v := range req.SetConfidence {
			setConf[k] = v
		}
	}

	var statusStr *string
	if req.Status != nil && *req.Status == pbuser.ResumeProfileStatus_CONFIRMED {
		s := "CONFIRMED"
		statusStr = &s
	} else if req.Status != nil && *req.Status == pbuser.ResumeProfileStatus_DRAFT {
		s := "DRAFT"
		statusStr = &s
	}

	version, err := h.resumeRepo.Patch(ctx, uint(req.UserId), patch, setConfirmed, setConf, statusStr)
	if err != nil {
		if err.Error() == "resume profile not found" {
			return nil, status.Errorf(codes.NotFound, "resume profile not found")
		}
		return nil, status.Errorf(codes.Internal, "patch resume profile: %v", err)
	}
	return &pbuser.PatchResumeProfileInternalResponse{Version: version}, nil
}

func (h *UserHandler) UpdateResumeProfile(ctx context.Context, req *pbuser.UpdateResumeProfileRequest) (*pbuser.UpdateResumeProfileResponse, error) {
	if req.Profile == nil {
		return nil, status.Errorf(codes.InvalidArgument, "profile required")
	}
	uid, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}
	userID := uint(uid)
	row, _ := h.resumeRepo.GetByUserID(ctx, userID)
	statusStr := "DRAFT"
	sourceMaterialID := ""
	if row != nil {
		statusStr = row.Status
		if row.SourceMaterialID != nil {
			sourceMaterialID = *row.SourceMaterialID
		}
	}
	areaIDs := make([]string, 0, len(req.Profile.Areas))
	for _, a := range req.Profile.Areas {
		areaIDs = append(areaIDs, a.Id)
	}
	var salaryMin *int
	if req.Profile.SalaryMin != nil {
		v := int(*req.Profile.SalaryMin)
		salaryMin = &v
	}
	conf, cf := map[string]interface{}{}, map[string]interface{}{}
	if row != nil {
		for k, v := range row.Confidence {
			conf[k] = v
		}
		for k, v := range row.ConfirmedFields {
			cf[k] = v
		}
	}
	_, err := h.resumeRepo.Upsert(ctx, userID, statusStr, sourceMaterialID,
		req.Profile.TargetRoles, req.Profile.ExperienceLevel, areaIDs, salaryMin, req.Profile.Currency,
		req.Profile.WorkFormat, req.Profile.SkillsTop, req.Profile.EducationLevel, req.Profile.Notes,
		conf, cf)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upsert resume profile")
	}
	return &pbuser.UpdateResumeProfileResponse{Success: true}, nil
}

func (h *UserHandler) RequestPasswordReset(ctx context.Context, req *pbuser.RequestPasswordResetRequest) (*pbuser.RequestPasswordResetResponse, error) {
	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}

	if err := h.passwordResetSvc.RequestPasswordReset(ctx, req.Email); err != nil {
		return nil, err
	}

	return &pbuser.RequestPasswordResetResponse{Ok: true}, nil
}

func (h *UserHandler) VerifyPasswordResetCode(ctx context.Context, req *pbuser.VerifyPasswordResetCodeRequest) (*pbuser.VerifyPasswordResetCodeResponse, error) {
	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	if req.Code == "" {
		return nil, status.Errorf(codes.InvalidArgument, "code is required")
	}

	valid, err := h.passwordResetSvc.VerifyCode(ctx, req.Email, req.Code)
	if err != nil {
		return nil, err
	}

	return &pbuser.VerifyPasswordResetCodeResponse{Valid: valid}, nil
}

func (h *UserHandler) ResetPassword(ctx context.Context, req *pbuser.ResetPasswordRequest) (*pbuser.ResetPasswordResponse, error) {
	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	if req.Code == "" {
		return nil, status.Errorf(codes.InvalidArgument, "code is required")
	}
	if req.NewPassword == "" {
		return nil, status.Errorf(codes.InvalidArgument, "new_password is required")
	}

	if err := h.passwordResetSvc.ResetPassword(ctx, req.Email, req.Code, req.NewPassword); err != nil {
		return nil, err
	}

	return &pbuser.ResetPasswordResponse{Ok: true}, nil
}

func (h *UserHandler) UpdateUserProfile(ctx context.Context, req *pbuser.UpdateUserProfileRequest) (*pbuser.UpdateUserProfileResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}

	var email string
	if req.Email != nil {
		email = *req.Email
		if email != "" {
			var otherID uint
			err := database.DB.QueryRow(ctx,
				"SELECT id FROM users WHERE email = $1 AND id != $2 LIMIT 1",
				email, userID).Scan(&otherID)
			if err == nil {
				return nil, status.Errorf(codes.AlreadyExists, "email already taken")
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, status.Errorf(codes.Internal, "database error")
			}
		}
	}

	firstName, lastName := "", ""
	if req.FirstName != nil {
		firstName = *req.FirstName
	}
	if req.LastName != nil {
		lastName = *req.LastName
	}

	// Собираем обновления по наличию полей
	updates := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argNum := 1
	if req.FirstName != nil {
		updates = append(updates, "first_name = $"+strconv.Itoa(argNum))
		args = append(args, firstName)
		argNum++
	}
	if req.LastName != nil {
		updates = append(updates, "last_name = $"+strconv.Itoa(argNum))
		args = append(args, lastName)
		argNum++
	}
	if req.Email != nil && *req.Email != "" {
		updates = append(updates, "email = $"+strconv.Itoa(argNum), "username = $"+strconv.Itoa(argNum))
		args = append(args, email, email)
		argNum += 2
	}
	if req.NotificationsEnabled != nil {
		updates = append(updates, "notifications_enabled = $"+strconv.Itoa(argNum))
		args = append(args, *req.NotificationsEnabled)
		argNum++
	}
	args = append(args, userID)
	query := "UPDATE users SET " + strings.Join(updates, ", ") + " WHERE id = $" + strconv.Itoa(argNum)
	_, err := database.DB.Exec(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update profile: %v", err)
	}

	return &pbuser.UpdateUserProfileResponse{Success: true}, nil
}

func (h *UserHandler) DeleteAccount(ctx context.Context, req *pbuser.DeleteAccountRequest) (*pbuser.DeleteAccountResponse, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not found in context")
	}

	if req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password is required")
	}

	err := h.accountDeletionSvc.DeleteAccount(ctx, userID, req.Password)
	if err != nil {
		return nil, err
	}

	return &pbuser.DeleteAccountResponse{Deleted: true}, nil
}
