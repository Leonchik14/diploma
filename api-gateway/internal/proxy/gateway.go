package proxy

import (
	"context"
	"log/slog"

	"api-gateway/internal/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pbauth "proto/auth"
	pbcalendar "proto/calendar"
	pbcoach "proto/coach"
	pbgateway "proto/gateway"
	pbjobs "proto/jobs"
	pbmaterials "proto/materials"
	pbuser "proto/user"
)

type GatewayProxy struct {
	pbgateway.UnimplementedBackendGatewayServer
	cfg     *config.Config
	logger  *slog.Logger
	authClient    pbauth.AuthServiceClient
	userClient    pbuser.UserServiceClient
	materialsClient pbmaterials.MaterialsServiceClient
	coachClient   pbcoach.CoachServiceClient
	jobsClient    pbjobs.JobsServiceClient
	calendarClient pbcalendar.CalendarServiceClient
}

func NewGatewayProxy(cfg *config.Config, logger *slog.Logger) *GatewayProxy {
	authConn := dial(cfg.UserServiceURL)
	userConn := dial(cfg.UserServiceURL)
	materialsConn := dial(cfg.MaterialsServiceURL)
	coachConn := dial(cfg.CoachServiceURL)
	jobsConn := dial(cfg.JobsServiceURL)
	calendarConn := dial(cfg.CalendarServiceURL)

	return &GatewayProxy{
		cfg:     cfg,
		logger:  logger,
		authClient:    pbauth.NewAuthServiceClient(authConn),
		userClient:    pbuser.NewUserServiceClient(userConn),
		materialsClient: pbmaterials.NewMaterialsServiceClient(materialsConn),
		coachClient:   pbcoach.NewCoachServiceClient(coachConn),
		jobsClient:    pbjobs.NewJobsServiceClient(jobsConn),
		calendarClient: pbcalendar.NewCalendarServiceClient(calendarConn),
	}
}

func dial(addr string) *grpc.ClientConn {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	return conn
}

func (p *GatewayProxy) forwardMetadata(ctx context.Context) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	return metadata.NewOutgoingContext(ctx, md)
}

// Auth
func (p *GatewayProxy) Register(ctx context.Context, req *pbauth.RegisterRequest) (*pbauth.RegisterResponse, error) {
	return p.authClient.Register(ctx, req)
}

func (p *GatewayProxy) Login(ctx context.Context, req *pbauth.LoginRequest) (*pbauth.LoginResponse, error) {
	return p.authClient.Login(ctx, req)
}

func (p *GatewayProxy) Refresh(ctx context.Context, req *pbauth.RefreshRequest) (*pbauth.RefreshResponse, error) {
	return p.authClient.Refresh(ctx, req)
}

func (p *GatewayProxy) CheckPasswordResetEmail(ctx context.Context, req *pbauth.PasswordResetCheckEmailRequest) (*pbauth.PasswordResetCheckEmailResponse, error) {
	return p.authClient.CheckPasswordResetEmail(ctx, req)
}

func (p *GatewayProxy) SendPasswordResetCode(ctx context.Context, req *pbauth.PasswordResetSendCodeRequest) (*pbauth.PasswordResetSendCodeResponse, error) {
	return p.authClient.SendPasswordResetCode(ctx, req)
}

func (p *GatewayProxy) VerifyPasswordReset(ctx context.Context, req *pbauth.PasswordResetVerifyRequest) (*pbauth.PasswordResetVerifyResponse, error) {
	return p.authClient.VerifyPasswordReset(ctx, req)
}

// User
func (p *GatewayProxy) GetMe(ctx context.Context, req *pbuser.GetMeRequest) (*pbuser.GetMeResponse, error) {
	return p.userClient.GetMe(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) GetResumeProfile(ctx context.Context, req *pbuser.GetResumeProfileRequest) (*pbuser.GetResumeProfileResponse, error) {
	return p.userClient.GetResumeProfile(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) UpdateResumeProfile(ctx context.Context, req *pbuser.UpdateResumeProfileRequest) (*pbuser.UpdateResumeProfileResponse, error) {
	return p.userClient.UpdateResumeProfile(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) UpdateUserProfile(ctx context.Context, req *pbuser.UpdateUserProfileRequest) (*pbuser.UpdateUserProfileResponse, error) {
	return p.userClient.UpdateUserProfile(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) UploadProfilePhoto(ctx context.Context, req *pbuser.UploadProfilePhotoRequest) (*pbuser.UploadProfilePhotoResponse, error) {
	return p.userClient.UploadProfilePhoto(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) GetProfilePhoto(ctx context.Context, req *pbuser.GetProfilePhotoRequest) (*pbuser.GetProfilePhotoResponse, error) {
	return p.userClient.GetProfilePhoto(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) GetResumeFile(ctx context.Context, req *pbuser.GetResumeFileRequest) (*pbuser.GetResumeFileResponse, error) {
	return p.userClient.GetResumeFile(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) DeleteAccount(ctx context.Context, req *pbuser.DeleteAccountRequest) (*pbuser.DeleteAccountResponse, error) {
	return p.userClient.DeleteAccount(p.forwardMetadata(ctx), req)
}

// Materials
func (p *GatewayProxy) UploadFile(ctx context.Context, req *pbmaterials.UploadFileRequest) (*pbmaterials.UploadFileResponse, error) {
	return p.materialsClient.UploadFile(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) DownloadFile(ctx context.Context, req *pbmaterials.DownloadFileRequest) (*pbmaterials.DownloadFileResponse, error) {
	return p.materialsClient.DownloadFile(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) ListFolder(ctx context.Context, req *pbmaterials.ListFolderRequest) (*pbmaterials.ListFolderResponse, error) {
	return p.materialsClient.ListFolder(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) CreateFolder(ctx context.Context, req *pbmaterials.CreateFolderRequest) (*pbmaterials.CreateFolderResponse, error) {
	return p.materialsClient.CreateFolder(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) CreateLink(ctx context.Context, req *pbmaterials.CreateLinkRequest) (*pbmaterials.CreateLinkResponse, error) {
	return p.materialsClient.CreateLink(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) RenameNode(ctx context.Context, req *pbmaterials.RenameNodeRequest) (*pbmaterials.RenameNodeResponse, error) {
	return p.materialsClient.RenameNode(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) DeleteNode(ctx context.Context, req *pbmaterials.DeleteNodeRequest) (*pbmaterials.DeleteNodeResponse, error) {
	return p.materialsClient.DeleteNode(p.forwardMetadata(ctx), req)
}

// Coach
func (p *GatewayProxy) Ask(ctx context.Context, req *pbcoach.AskRequest) (*pbcoach.AskResponse, error) {
	return p.coachClient.Ask(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) ParseResume(ctx context.Context, req *pbcoach.ParseResumeRequest) (*pbcoach.ParseResumeResponse, error) {
	return p.coachClient.ParseResume(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) UploadAndParseResume(ctx context.Context, req *pbcoach.UploadAndParseResumeRequest) (*pbcoach.UploadAndParseResumeResponse, error) {
	return p.coachClient.UploadAndParseResume(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) AnswerResume(ctx context.Context, req *pbcoach.AnswerResumeRequest) (*pbcoach.AnswerResumeResponse, error) {
	return p.coachClient.AnswerResume(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) GetResumeSession(ctx context.Context, req *pbcoach.GetResumeSessionRequest) (*pbcoach.GetResumeSessionResponse, error) {
	return p.coachClient.GetResumeSession(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) PrepareForVacancy(ctx context.Context, req *pbcoach.PrepareForVacancyRequest) (*pbcoach.PrepareForVacancyResponse, error) {
	return p.coachClient.PrepareForVacancy(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) ReviewResume(ctx context.Context, req *pbcoach.ReviewResumeRequest) (*pbcoach.ReviewResumeResponse, error) {
	return p.coachClient.ReviewResume(p.forwardMetadata(ctx), req)
}

// Jobs
func (p *GatewayProxy) SearchJobs(ctx context.Context, req *pbjobs.SearchJobsRequest) (*pbjobs.SearchJobsResponse, error) {
	return p.jobsClient.SearchJobs(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) AddFavorite(ctx context.Context, req *pbjobs.AddFavoriteRequest) (*pbjobs.AddFavoriteResponse, error) {
	return p.jobsClient.AddFavorite(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) RemoveFavorite(ctx context.Context, req *pbjobs.RemoveFavoriteRequest) (*pbjobs.RemoveFavoriteResponse, error) {
	return p.jobsClient.RemoveFavorite(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) ListFavorites(ctx context.Context, req *pbjobs.ListFavoritesRequest) (*pbjobs.ListFavoritesResponse, error) {
	return p.jobsClient.ListFavorites(p.forwardMetadata(ctx), req)
}

// Calendar
func (p *GatewayProxy) CreateEvent(ctx context.Context, req *pbcalendar.CreateEventRequest) (*pbcalendar.CreateEventResponse, error) {
	return p.calendarClient.CreateEvent(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) GetEvent(ctx context.Context, req *pbcalendar.GetEventRequest) (*pbcalendar.GetEventResponse, error) {
	return p.calendarClient.GetEvent(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) UpdateEvent(ctx context.Context, req *pbcalendar.UpdateEventRequest) (*pbcalendar.UpdateEventResponse, error) {
	return p.calendarClient.UpdateEvent(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) DeleteEvent(ctx context.Context, req *pbcalendar.DeleteEventRequest) (*pbcalendar.DeleteEventResponse, error) {
	return p.calendarClient.DeleteEvent(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) ListEvents(ctx context.Context, req *pbcalendar.ListEventsRequest) (*pbcalendar.ListEventsResponse, error) {
	return p.calendarClient.ListEvents(p.forwardMetadata(ctx), req)
}

func (p *GatewayProxy) ListUpcoming(ctx context.Context, req *pbcalendar.ListUpcomingRequest) (*pbcalendar.ListUpcomingResponse, error) {
	return p.calendarClient.ListUpcoming(p.forwardMetadata(ctx), req)
}
