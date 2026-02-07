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

type GetResumeProfileResult struct {
	Profile *ResumeProfile
	Status  string // "DRAFT" or "CONFIRMED"
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

func (c *UserClient) GetResumeProfile(ctx context.Context, userID string) (*GetResumeProfileResult, error) {
	uid, err := parseUserID(userID)
	if err != nil {
		return nil, err
	}
	md := metadata.New(map[string]string{
		"x-internal-api-key": c.internalKey,
		"x-user-id":          userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &pbuser.GetResumeProfileInternalRequest{UserId: uid}
	resp, err := c.client.GetResumeProfileInternal(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	if resp.Profile == nil {
		return nil, fmt.Errorf("profile is nil")
	}

	profile := &ResumeProfile{
		TargetRoles: resp.Profile.TargetRoles,
		SkillsTop:   resp.Profile.SkillsTop,
		WorkFormat:  resp.Profile.WorkFormat,
	}

	if resp.Profile.ExperienceLevel != nil {
		profile.ExperienceLevel = resp.Profile.ExperienceLevel
	}
	if resp.Profile.SalaryMin != nil {
		profile.SalaryMin = resp.Profile.SalaryMin
	}
	if resp.Profile.Currency != nil {
		profile.Currency = resp.Profile.Currency
	}

	for _, pbArea := range resp.Profile.Areas {
		profile.Areas = append(profile.Areas, Area{
			ID:   pbArea.Id,
			Name: pbArea.Name,
		})
	}

	statusStr := "DRAFT"
	if resp.Status == pbuser.ResumeProfileStatus_CONFIRMED {
		statusStr = "CONFIRMED"
	}
	return &GetResumeProfileResult{Profile: profile, Status: statusStr}, nil
}

func parseUserID(s string) (uint32, error) {
	var u uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid user id: %s", s)
		}
		u = u*10 + uint64(c-'0')
		if u > 0xFFFFFFFF {
			return 0, fmt.Errorf("user id overflow: %s", s)
		}
	}
	return uint32(u), nil
}

func (c *UserClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
