package clients

import (
	"context"
	"log/slog"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pbcoach "proto/coach"
)

type CoachClient struct {
	client      pbcoach.CoachServiceClient
	conn        *grpc.ClientConn
	internalKey string
	logger      *slog.Logger
}

func NewCoachClient(addr, internalKey string, logger *slog.Logger) *CoachClient {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("Failed to connect to coach-service", "error", err)
		return nil
	}

	return &CoachClient{
		client:      pbcoach.NewCoachServiceClient(conn),
		conn:        conn,
		internalKey: internalKey,
		logger:      logger,
	}
}

func (c *CoachClient) DeleteUserData(ctx context.Context, userID uint) error {
	md := metadata.Pairs("x-internal-api-key", c.internalKey, "x-user-id", strconv.FormatUint(uint64(userID), 10))
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := c.client.DeleteUserData(ctx, &pbcoach.DeleteUserDataRequest{
		UserId: uint32(userID),
	})
	return err
}
