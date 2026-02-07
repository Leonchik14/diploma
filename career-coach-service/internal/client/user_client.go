package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pbuser "proto/user"
)

type ResumeProfile struct {
	TargetRoles     []string
	ExperienceLevel *string
	Areas           []Area
	SalaryMin       *float64
	Currency        *string
	WorkFormat      []string
	SkillsTop       []string
}

type Area struct {
	ID   string
	Name string
}

type UserClient struct {
	client      pbuser.UserServiceClient
	conn        *grpc.ClientConn
	internalKey string
}

func NewUserClient(grpcAddr, internalKey string, timeout time.Duration) *UserClient {
	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to user-service: %v", err))
	}
	return &UserClient{
		client:      pbuser.NewUserServiceClient(conn),
		conn:        conn,
		internalKey: internalKey,
	}
}

func (c *UserClient) withInternalMD(ctx context.Context) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
		"x-internal-api-key": c.internalKey,
	}))
}

func (c *UserClient) withInternalMDAndUser(ctx context.Context, userID uint) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
		"x-internal-api-key": c.internalKey,
		"x-user-id":          fmt.Sprintf("%d", userID),
	}))
}

func (c *UserClient) UpsertResumeProfileInternal(ctx context.Context, userID uint, sourceMaterialID string, profile *pbuser.ResumeProfile, status pbuser.ResumeProfileStatus, confirmedFields map[string]bool, confidence map[string]float64) (int64, error) {
	ctx = c.withInternalMD(ctx)
	resp, err := c.client.UpsertResumeProfileInternal(ctx, &pbuser.UpsertResumeProfileInternalRequest{
		UserId:           uint32(userID),
		SourceMaterialId:  sourceMaterialID,
		Profile:           profile,
		Status:            status,
		ConfirmedFields:   confirmedFields,
		Confidence:        confidence,
	})
	if err != nil {
		return 0, err
	}
	return resp.Version, nil
}

func (c *UserClient) PatchResumeProfileInternal(ctx context.Context, userID uint, patch *pbuser.ResumeProfilePatch, setConfirmedFields map[string]bool, setConfidence map[string]float64, status *pbuser.ResumeProfileStatus) (int64, error) {
	ctx = c.withInternalMD(ctx)
	req := &pbuser.PatchResumeProfileInternalRequest{
		UserId:             uint32(userID),
		Patch:              patch,
		SetConfirmedFields: setConfirmedFields,
		SetConfidence:      setConfidence,
		Status:             status,
	}
	resp, err := c.client.PatchResumeProfileInternal(ctx, req)
	if err != nil {
		return 0, err
	}
	return resp.Version, nil
}

func (c *UserClient) GetResumeProfileInternal(ctx context.Context, userID uint) (*pbuser.GetResumeProfileInternalResponse, error) {
	ctx = c.withInternalMD(ctx)
	return c.client.GetResumeProfileInternal(ctx, &pbuser.GetResumeProfileInternalRequest{
		UserId: uint32(userID),
	})
}

func (c *UserClient) GetResumeProfile(ctx context.Context, userID uint) (*ResumeProfile, error) {
	resp, err := c.GetResumeProfileInternal(ctx, userID)
	if err != nil {
		return nil, err
	}
	if resp.Profile == nil {
		return nil, fmt.Errorf("profile is nil")
	}
	p := resp.Profile
	out := &ResumeProfile{
		TargetRoles: p.TargetRoles,
		SkillsTop:   p.SkillsTop,
		WorkFormat:  p.WorkFormat,
	}
	if p.ExperienceLevel != nil {
		out.ExperienceLevel = p.ExperienceLevel
	}
	if p.SalaryMin != nil {
		out.SalaryMin = p.SalaryMin
	}
	if p.Currency != nil {
		out.Currency = p.Currency
	}
	for _, a := range p.Areas {
		out.Areas = append(out.Areas, Area{ID: a.Id, Name: a.Name})
	}
	return out, nil
}

func (c *UserClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
